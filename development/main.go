package main

import (
	"net/http"
	//"io/ioutil"
	"log"
	"html/template"
	"database/sql"
	//_ "github.com/mattn/go-sqlite3"
	_ "github.com/mxk/go-sqlite/sqlite3"
	"strconv"
	"strings"
	"time"
	"os"
	"runtime"
	"os/exec"
	"fmt"
)

var mysql_user string = "root"
var mysql_pass string = "aelo"
var mysql_db string = "mandy"

var errDefault = 0
var errTimeConversion = 3
var errMysqlDBname = 1
var errDBquery = 2
var errBrowser = 4

var files map[string]string

var dbName = "mandy"

var footer = `
<div class="footer">
	    <div>
		<strong>Copyright</strong> aelo Software Solutions &copy; 2017-2027
	    </div>
</div>
`

type van struct {
	Id      int64
	Des     string
	Loads   []loading
	Unloads []unloading
	Pro     []productVan

	Delete  bool
	Active  bool
}

type loading struct {
	Id    int64
	V_id  int64
	P_id  int64
	Qty   int64
	Dte   string
	Pro   product

	P_des string
}

type unloading struct {
	Id    int64
	V_id  int64
	P_id  int64
	Qty   int64
	Dte   string
	Pro   product

	P_des string
}

type productVan struct {
	Id       int64
	Des      string
	S_p      int64
	B_p      int64
	Qty      int64

	Loaded   int64
	Unloaded int64
	Rest     int64
}

func getProductVan(id string) productVan {
	tmp := productVan{}
	rows := getResultDB("SELECT * FROM pro WHERE id=" + id)

	for rows.Next() {

		var id sql.NullInt64
		var des sql.NullString
		var s_p sql.NullInt64
		var b_p sql.NullInt64

		err := rows.Scan(&id, &des, &s_p, &b_p)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String
		tmp.S_p = s_p.Int64
		tmp.B_p = b_p.Int64

		rows2 := getResultDB("select sum(qty)- (select sum(qty) from inv_reg where p_id=" + strconv.FormatInt(tmp.Id, 10) + ") as qty from grn_reg where p_id=" + strconv.FormatInt(tmp.Id, 10))
		rows2.Next()
		var qty sql.NullInt64
		err = rows2.Scan(&qty)
		checkErr(err, errDBquery)

		rows2.Close()
		tmp.Qty = qty.Int64
	}
	rows.Close()
	return tmp
}

func getVansForDelivery(filter string, val string) []van {
	tmp2 := []van{}

	rows := getResultDB("SELECT * FROM van WHERE " + filter + "=" + val)
	i := 0
	for rows.Next() {
		tmp := van{}

		if i == 0 {
			tmp.Active = true
		} else {
			tmp.Active = false
		}
		i++
		var id sql.NullInt64
		var des sql.NullString

		err := rows.Scan(&id, &des)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String
		tmp.Delete = true
		rows2 := getResultDB("select p_id , sum(qty) as qty from ldng where v_id = " + strconv.FormatInt(tmp.Id, 10) + "  group by p_id")
		for rows2.Next() {
			tmp.Delete = false
			var p_id sql.NullInt64
			var qty sql.NullInt64

			err := rows2.Scan(&p_id, &qty)
			checkErr(err, errDBquery)

			tmp3 := getProductVan(strconv.FormatInt(p_id.Int64, 10))
			tmp3.Loaded = qty.Int64
			//select p_id , sum(qty) as qty from u_ldng where v_id = 0 and p_id = 0 group by p_id
			rows3 := getResultDB("select sum(qty) as qty from u_ldng where v_id = " + strconv.FormatInt(tmp.Id, 10) + " and p_id = " + strconv.FormatInt(p_id.Int64, 10) + " group by p_id")
			rows3.Next()

			err = rows3.Scan(&qty)
			if err != nil {
				qty.Int64 = 0
			}

			tmp3.Unloaded = qty.Int64

			tmp3.Rest = tmp3.Loaded - tmp3.Unloaded
			rows3.Close()


			//to calculate sold items
			rows3 = getResultDB("select sum(qty) as qty from inv_reg LEFT JOIN inv ON inv_reg.i_id=inv.id where inv.v_id = " + strconv.FormatInt(tmp.Id, 10) + " and inv_reg.p_id = " + strconv.FormatInt(p_id.Int64, 10) + " group by inv_reg.p_id")
			rows3.Next()

			err = rows3.Scan(&qty)
			if err != nil {
				qty.Int64 = 0
			}

			tmp3.Rest -= qty.Int64
			rows3.Close()

			tmp.Pro = append(tmp.Pro, tmp3)
		}
		rows2.Close()
		tmp2 = append(tmp2, tmp)
	}

	return tmp2
}

func getLoadings(van string) []loading {
	tmp2 := []loading{}
	rows := getResultDB("SELECT * FROM ldng WHERE v_id=" + van + " ORDER BY dte DESC")

	for rows.Next() {
		tmp := loading{}

		var id sql.NullInt64
		var v_id sql.NullInt64
		var p_id sql.NullInt64
		var qty sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &v_id, &p_id, &qty, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Qty = qty.Int64
		tmp.Dte = string(dte)

		rows2 := getResultDB("select des from pro where id = " + strconv.FormatInt(p_id.Int64, 10))
		var des sql.NullString
		for rows2.Next() {
			err := rows2.Scan(&des)
			checkErr(err, errDBquery)
		}
		tmp.P_des = des.String
		rows2.Close()
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func getUnoadings(van string) []unloading {
	tmp2 := []unloading{}
	rows := getResultDB("SELECT * FROM u_ldng WHERE v_id=" + van + " ORDER BY dte DESC")

	for rows.Next() {
		tmp := unloading{}

		var id sql.NullInt64
		var v_id sql.NullInt64
		var p_id sql.NullInt64
		var qty sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &v_id, &p_id, &qty, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Qty = qty.Int64
		tmp.Dte = string(dte)

		rows2 := getResultDB("select des from pro where id = " + strconv.FormatInt(p_id.Int64, 10))
		var des sql.NullString
		for rows2.Next() {
			err := rows2.Scan(&des)
			checkErr(err, errDBquery)
		}
		tmp.P_des = des.String
		rows2.Close()
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func getVansForLoading(filter string, val string) []van {
	tmp2 := []van{}

	rows := getResultDB("SELECT * FROM van WHERE " + filter + "=" + val)
	i := 0
	for rows.Next() {
		tmp := van{}

		if i == 0 {
			tmp.Active = true
		} else {
			tmp.Active = false
		}
		i++
		var id sql.NullInt64
		var des sql.NullString

		err := rows.Scan(&id, &des)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String

		tmp.Loads = getLoadings(strconv.FormatInt(id.Int64, 10))

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func getVansForUnoading(filter string, val string) []van {
	tmp2 := []van{}

	rows := getResultDB("SELECT * FROM van WHERE " + filter + "=" + val)
	i := 0
	for rows.Next() {
		tmp := van{}

		if i == 0 {
			tmp.Active = true
		} else {
			tmp.Active = false
		}
		i++
		var id sql.NullInt64
		var des sql.NullString

		err := rows.Scan(&id, &des)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String

		tmp.Unloads = getUnoadings(strconv.FormatInt(id.Int64, 10))

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

var debugCount = 0

func debugMSG(msg string) {
	println(strconv.Itoa(debugCount) + " Done!")
	debugCount++
	//println(msg)
}

func readFile(path string) string {
	s := ""

	//out, _ := ioutil.ReadFile("./core/" + path)
	//s = string(out)

	s = files[path]

	return s
}

var homepage = true

func showFile(w http.ResponseWriter, r *http.Request, file string, data interface{}) {
	if !checkUser(w, r) {
		return
	}
	fm := template.FuncMap{"dec": func(a int64) string {
		return int2floatStr(a)
	}}
	t := template.New("fieldname example")
	if homepage {
		homepage = false
		t, _ = t.Parse(readFile("home"))
	} else {
		t, _ = t.Funcs(fm).Parse(readFile(file))
	}
	t.Execute(w, data)
}

func checkErr(err error, typ int) {
	// Of course, this name isn't unique,
	// I usually use time.Now().Unix() or something
	// to get unique log names.

	if err == nil {
		return
	}
	switch typ {
	default:
		println("Error occured!, Please contact developer :)")
		t := time.Now().Format("01_02_2006_15.04.05")
		logFile, err := os.Create("./log/" + t + ".txt")
		log.SetOutput(logFile)
		log.Panic(err)
	}
}

func initDatabase() {
	executeDB(readFile("db.sql"))
}

func getResultDB(query string) *sql.Rows {
	debugMSG(query)
	database, _ := sql.Open("sqlite3", "./database/" + dbName + ".db")
	rows, err := database.Query(query)
	checkErr(err, errDBquery)

	err = database.Close()
	checkErr(err, errDBquery)
	database.Close()
	return rows
}

func executeDB(exe string) {
	debugMSG(exe)
	database, _ := sql.Open("sqlite3", "./database/" + dbName + ".db")
	_, err := database.Exec(exe)
	checkErr(err, errDBquery)
	database.Close()
}

func insertData(table string, vals string) {
	executeDB("INSERT INTO " + table + " VALUES(" + vals + ")")
}

func deleteData(table string, id string) {
	executeDB("DELETE FROM " + table + " WHERE id = " + id)
}

func updateData(table string, id string, str string) {
	executeDB("UPDATE " + table + " SET " + str + " WHERE id=" + id)
}

func getNextID(table string) int64 {
	rows := getResultDB("select max(id) + 1 as newid from " + table)

	if rows == nil {
		return -1
	}

	rows.Next()

	var newid sql.NullInt64

	err := rows.Scan(&newid)
	checkErr(err, errDBquery)

	rows.Close()

	return newid.Int64
}

func checkUser(w http.ResponseWriter, r *http.Request) bool {
	return true
}

type vanInvoice struct {
	Id       int64
	Des      string
	Invoices []_invoice

	Active   bool
}

func getVansForInvoice(filter string, val string) []vanInvoice {
	tmp2 := []vanInvoice{}

	rows := getResultDB("SELECT * FROM van WHERE " + filter + "=" + val)
	i := 0
	for rows.Next() {
		tmp := vanInvoice{}

		if i == 0 {
			tmp.Active = true
		} else {
			tmp.Active = false
		}
		i++
		var id sql.NullInt64
		var des sql.NullString

		err := rows.Scan(&id, &des)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String

		tmp.Invoices = get_invoices("v_id", strconv.FormatInt(id.Int64, 10))

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func getSimplyCustomers(filter string, val string) [] customer {
	tmp2 := []customer{}

	rows := getResultDB("SELECT * FROM cus WHERE " + filter + "=" + val)

	for rows.Next() {
		tmp := customer{}

		var id sql.NullInt64
		var name sql.NullString
		var phn sql.NullString
		var ad sql.NullString

		err := rows.Scan(&id, &name, &phn, &ad)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Name = name.String
		tmp.Phn = phn.String
		tmp.Ad = ad.String

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func invoice(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.Form.Get("submit") == "Add" {
		//m,d,y
		s := strings.Split(r.Form.Get("dte"), "/")
		dte := s[2] + "-" + s[0] + "-" + s[1]
		executeDB("INSERT INTO inv VALUES(" + r.Form.Get("id") + "," + r.Form.Get("c_id") + "," + r.Form.Get("v_id") + ",'" + r.Form.Get("i_no") + "','" + r.Form.Get("po_no") + "',0,'" + dte + "')")
		http.Redirect(w, r, "editInvoice?id=" + r.Form.Get("id"), http.StatusSeeOther)
		return
	}

	type data struct {
		Title string
		Cus   []customer
		Dte   string
		NxtID int64
		Vans  []vanInvoice
	}
	now := time.Now()

	result := data{dbName, getSimplyCustomers("''", "''"), now.Format("01/02/2006"), getNextID("inv"), getVansForInvoice("''", "''")}

	showFile(w, r, "invoice", result)
}

func grn(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.Form.Get("submit") == "Add" {
		//m,d,y
		s := strings.Split(r.Form.Get("dte"), "/")
		dte := s[2] + "-" + s[0] + "-" + s[1]
		executeDB("INSERT INTO grn VALUES(" + r.Form.Get("id") + "," + r.Form.Get("v_id") + ",'" + r.Form.Get("g_no") + "',0,'" + dte + "')")
		http.Redirect(w, r, "editGRN?id=" + r.Form.Get("id"), http.StatusSeeOther)
		return
	}

	type data struct {
		Title   string
		Vendors []vendor
		Grns    []_grn
		Dte     string
		NxtID   int64
	}
	now := time.Now()

	result := data{dbName, getSimplyVendors("''", "''"), get_grns("''", "''"), now.Format("01/02/2006"), getNextID("grn")}
	showFile(w, r, "grn", result)
}

type customerPayment struct {
	Id     int64
	Dte    string
	I_id   int64
	Des    string
	Tot    int64

	I_no   string
	C_name string
}

func getCustomerPayments(filter string, val string) []customerPayment {
	tmp2 := []customerPayment{}

	rows := getResultDB("SELECT * FROM cus_pay WHERE " + filter + "=" + val + "  ORDER BY dte DESC")

	for rows.Next() {
		tmp := customerPayment{}
		var id sql.NullInt64
		var dte sql.RawBytes
		var i_id sql.NullInt64
		var tot sql.NullInt64
		var des sql.NullString

		err := rows.Scan(&id, &dte, &i_id, &des, &tot)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Dte = string(dte)
		tmp.I_id = i_id.Int64
		tmp.Des = des.String
		tmp.Tot = tot.Int64

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

type invoiceRecord struct {
	Id      int64
	I_id    int64
	P_id    int64
	P_des   string
	B_p     int64
	S_p     int64
	Margine float64
	Qty     int64
	Tot     int64
}

func calculateMargine(bp float64, sp float64) float64 {
	if bp == 0.0 {
		return 0.0
	}
	return ((sp - bp) * 100) / (bp)
}

func getInvoiceRecords(filter string, val string) [] invoiceRecord {
	tmp2 := []invoiceRecord{}

	rows := getResultDB("SELECT * FROM inv_reg WHERE " + filter + "=" + val)

	for rows.Next() {
		tmp := invoiceRecord{}
		var id sql.NullInt64
		var i_id sql.NullInt64
		var p_id sql.NullInt64
		var b_p sql.NullInt64
		var s_p sql.NullInt64
		var qty sql.NullInt64

		err := rows.Scan(&id, &i_id, &p_id, &b_p, &s_p, &qty)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.I_id = i_id.Int64
		tmp.P_id = p_id.Int64
		tmp.B_p = b_p.Int64
		tmp.S_p = s_p.Int64
		tmp.Qty = qty.Int64

		rows2 := getResultDB("SELECT des FROM pro WHERE id=" + strconv.FormatInt(tmp.P_id, 10))
		rows2.Next()
		var des sql.NullString
		err = rows2.Scan(&des)
		checkErr(err, errDBquery)

		rows2.Close()

		tmp.P_des = des.String

		tmp.Margine = calculateMargine(float64(tmp.B_p), float64(tmp.S_p))

		tmp.Tot = tmp.S_p * tmp.Qty
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

type _invoice struct {
	Id               int64
	C_id             int64
	V_id             int64
	I_no             string
	Po_no            string
	Vat              int64
	Sub_tot          int64
	Grnd_tot         int64
	Dte              string
	PaymentsDone     int64
	RemainingPayment int64
	Progress         int64
	Margine          float64
	DeleteBTN        bool
	Records          []invoiceRecord
	Payments         []customerPayment

	Cus              customer
}

func getSimplyCustomer(id string) customer {
	tmp := customer{}

	rows := getResultDB("SELECT * FROM cus WHERE id=" + id)

	for rows.Next() {

		var id sql.NullInt64
		var name sql.NullString
		var phn sql.NullString
		var ad sql.NullString

		err := rows.Scan(&id, &name, &phn, &ad)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Name = name.String
		tmp.Phn = phn.String
		tmp.Ad = ad.String
	}
	rows.Close()
	return tmp
}

func get_invoices(filter string, val string) [] _invoice {
	tmp2 := []_invoice{}

	rows := getResultDB("SELECT * FROM inv WHERE " + filter + "=" + val + " ORDER BY dte DESC")

	for rows.Next() {
		tmp := _invoice{}

		var id sql.NullInt64
		var c_id sql.NullInt64
		var v_id sql.NullInt64
		var i_no sql.NullString
		var po_no sql.NullString
		var vat sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &c_id, &v_id, &i_no, &po_no, &vat, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.C_id = c_id.Int64
		tmp.V_id = v_id.Int64
		tmp.I_no = i_no.String
		tmp.Po_no = po_no.String
		tmp.Vat = vat.Int64
		tmp.Dte = string(dte)

		tmp.Records = getInvoiceRecords("i_id", strconv.FormatInt(id.Int64, 10))
		tmp.Payments = getCustomerPayments("i_id", strconv.FormatInt(id.Int64, 10))

		tmp.Cus = getSimplyCustomer(strconv.FormatInt(c_id.Int64, 10))

		tmp.PaymentsDone = 0
		for _, p := range tmp.Payments {
			tmp.PaymentsDone += p.Tot
		}

		tmp.Margine = 0.0
		tmp.Sub_tot = 0
		for _, p := range tmp.Records {
			tmp.Margine += p.Margine
			tmp.Sub_tot += p.Tot
		}
		tmp.Margine /= float64(len(tmp.Records))

		tmp.Grnd_tot = tmp.Vat + tmp.Sub_tot

		tmp.RemainingPayment = tmp.Grnd_tot - tmp.PaymentsDone
		if tmp.Grnd_tot == 0 {
			tmp.Progress = 100
		} else {
			tmp.Progress = (tmp.PaymentsDone * 100) / tmp.Grnd_tot
		}

		if len(tmp.Records) > 0 {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func get_invoice(id string) _invoice {
	tmp := _invoice{}

	rows := getResultDB("SELECT * FROM inv WHERE id=" + id)

	for rows.Next() {

		var id sql.NullInt64
		var c_id sql.NullInt64
		var v_id sql.NullInt64
		var i_no sql.NullString
		var po_no sql.NullString
		var vat sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &c_id, &v_id, &i_no, &po_no, &vat, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.C_id = c_id.Int64
		tmp.V_id = v_id.Int64
		tmp.I_no = i_no.String
		tmp.Po_no = po_no.String
		tmp.Vat = vat.Int64
		tmp.Dte = string(dte)

		tmp.Records = getInvoiceRecords("i_id", strconv.FormatInt(id.Int64, 10))
		tmp.Payments = getCustomerPayments("i_id", strconv.FormatInt(id.Int64, 10))

		tmp.Cus = getSimplyCustomer(strconv.FormatInt(c_id.Int64, 10))

		tmp.PaymentsDone = 0
		for _, p := range tmp.Payments {
			tmp.PaymentsDone += p.Tot
		}

		tmp.Margine = 0.0
		tmp.Sub_tot = 0
		for _, p := range tmp.Records {
			tmp.Margine += p.Margine
			tmp.Sub_tot += p.Tot
		}
		tmp.Margine /= float64(len(tmp.Records))

		tmp.Grnd_tot = tmp.Vat + tmp.Sub_tot

		tmp.RemainingPayment = tmp.Grnd_tot - tmp.PaymentsDone
		if tmp.Grnd_tot == 0 {
			tmp.Progress = 100
		} else {
			tmp.Progress = (tmp.PaymentsDone * 100) / tmp.Grnd_tot
		}

		if len(tmp.Records) > 0 {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}

	}
	rows.Close()
	return tmp
}

type customer struct {
	Id        int64
	Name      string
	Phn       string
	Ad        string
	Due       int64
	Dne       int64
	Pro       int64
	DeleteBTN bool
	Active    bool
	Invoices  []_invoice
}

func getCustomers(filter string, val string) [] customer {
	tmp2 := []customer{}

	rows := getResultDB("SELECT * FROM cus WHERE " + filter + "=" + val + "ORDER BY name")
	i := true
	for rows.Next() {
		tmp := customer{}
		tmp.Active = i
		i = false
		var id sql.NullInt64
		var name sql.NullString
		var phn sql.NullString
		var ad sql.NullString

		err := rows.Scan(&id, &name, &phn, &ad)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Name = name.String
		tmp.Phn = phn.String
		tmp.Ad = ad.String

		tmp.Invoices = get_invoices("c_id", strconv.FormatInt(id.Int64, 10))

		tmp.DeleteBTN = true
		tmp.Dne = 0;
		tmp.Due = 0;
		for _, i := range tmp.Invoices {
			tmp.Dne += i.PaymentsDone
			tmp.Due += i.RemainingPayment
		}

		if len(tmp.Invoices) > 0 {
			tmp.DeleteBTN = false
		}

		if (tmp.Due <= 0 && tmp.Dne <= 0) {
			tmp.Pro = 100
		} else {
			tmp.Pro = (tmp.Dne * 100) / (tmp.Due + tmp.Dne)
		}
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func customers(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.Form.Get("submit") == "Add" {
		insertData("cus", r.Form.Get("id") + ",'" + r.Form.Get("name") + "','" + r.Form.Get("phn") + "','" + r.Form.Get("ad") + "'")
	} else if r.Form.Get("submit") == "Save" {
		updateData("cus", r.Form.Get("id"), "name='" + r.Form.Get("name") + "', phn='" + r.Form.Get("phn") + "', ad='" + r.Form.Get("ad") + "'")
	} else if r.Form.Get("submit") == "Delete" {
		deleteData("cus", r.Form.Get("id"))
	}

	type sendData struct {
		Title     string
		Nid       int64
		Customers []customer
	}

	results := sendData{dbName, getNextID("cus"), getCustomers("''", "''")}

	showFile(w, r, "customers", results)
}

type grnRecord struct {
	Id    int64
	G_id  int64
	P_id  int64
	P_des string
	B_p   int64
	Qty   int64
	Tot   int64
}

func getGrnRecords(filter string, val string) [] grnRecord {
	tmp2 := []grnRecord{}

	rows := getResultDB("SELECT * FROM grn_reg WHERE " + filter + "=" + val)

	for rows.Next() {
		tmp := grnRecord{}
		var id sql.NullInt64
		var g_id sql.NullInt64
		var p_id sql.NullInt64
		var b_p sql.NullInt64
		var qty sql.NullInt64

		err := rows.Scan(&id, &g_id, &p_id, &b_p, &qty)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.G_id = g_id.Int64
		tmp.P_id = p_id.Int64
		tmp.B_p = b_p.Int64
		tmp.Qty = qty.Int64

		rows2 := getResultDB("SELECT des FROM pro WHERE id=" + strconv.FormatInt(tmp.P_id, 10))
		rows2.Next()
		var des sql.NullString
		err = rows2.Scan(&des)
		checkErr(err, errDBquery)

		tmp.P_des = des.String

		tmp.Tot = tmp.B_p * tmp.Qty
		rows2.Close()
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

type _grn struct {
	Id        int64
	V_id      int64
	G_no      string
	Vat       int64
	Sub_tot   int64
	Grnd_tot  int64
	Dte       string
	DeleteBTN bool
	Ven       vendor
	Records   []grnRecord
}

func get_grns(filter string, val string) [] _grn {
	tmp2 := []_grn{}

	rows := getResultDB("SELECT * FROM grn WHERE " + filter + "=" + val + " ORDER BY dte DESC")

	for rows.Next() {
		tmp := _grn{}

		var id sql.NullInt64
		var v_id sql.NullInt64
		var g_no sql.NullString
		var vat sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &v_id, &g_no, &vat, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.V_id = v_id.Int64
		tmp.G_no = g_no.String
		tmp.Vat = vat.Int64
		tmp.Dte = string(dte)

		tmp.Records = getGrnRecords("g_id", strconv.FormatInt(id.Int64, 10))

		tmp.Sub_tot = 0
		for _, g := range tmp.Records {
			tmp.Sub_tot += g.Tot
		}

		tmp.Grnd_tot = tmp.Vat + tmp.Sub_tot

		if len(tmp.Records) > 0 {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}

		tmp.Ven = getVendor(strconv.FormatInt(v_id.Int64, 10))

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func get_grn(id string) _grn {

	rows := getResultDB("SELECT * FROM grn WHERE id=" + id)
	tmp := _grn{}
	for rows.Next() {

		var id sql.NullInt64
		var v_id sql.NullInt64
		var g_no sql.NullString
		var vat sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &v_id, &g_no, &vat, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.V_id = v_id.Int64
		tmp.G_no = g_no.String
		tmp.Vat = vat.Int64
		tmp.Dte = string(dte)

		tmp.Records = getGrnRecords("g_id", strconv.FormatInt(id.Int64, 10))

		tmp.Sub_tot = 0
		for _, g := range tmp.Records {
			tmp.Sub_tot += g.Tot
		}

		tmp.Grnd_tot = tmp.Vat + tmp.Sub_tot

		if len(tmp.Records) > 0 {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}

		tmp.Ven = getVendor(strconv.FormatInt(v_id.Int64, 10))
	}
	rows.Close()
	return tmp
}

func editGRN(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	type data struct {
		Title    string
		Grn      _grn
		Products []product
		NxtID    int64
	}

	switch r.Form.Get("submit") {
	case "Save":
		executeDB("UPDATE grn SET vat=" + strfloat2strint(r.Form.Get("vat")) + " WHERE id=" + r.Form.Get("id"))
	case "remove":
		executeDB("DELETE FROM grn_reg WHERE id=" + r.Form.Get("r_id"))
	case "Add":
		s := strings.Split(r.Form.Get("p_id"), ",")
		p_id := s[0]
		executeDB("INSERT INTO grn_reg VALUES(" + r.Form.Get("r_id") + "," + r.Form.Get("id") + "," + p_id + "," + strfloat2strint(r.Form.Get("b_p")) + "," + r.Form.Get("qty") + ")")
	//println(r.Form.Get("p_id"))
	case "Delete":
		executeDB("DELETE FROM grn_reg WHERE g_id=" + r.Form.Get("id"))
		deleteData("grn", r.Form.Get("id"))
		http.Redirect(w, r, "grn", http.StatusSeeOther)
	}

	if r.Form.Get("id") == "" {
		http.Redirect(w, r, "grn", http.StatusSeeOther)
		return
	}

	result := data{dbName, get_grn(r.Form.Get("id")), getProducts("''", "''"), getNextID("grn_reg")}
	showFile(w, r, "editGRN", result)
}

func editInvoice(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	type data struct {
		Title    string
		Inv      _invoice
		Products []productVan
		NxtID    int64
	}

	switch r.Form.Get("submit") {
	case "Save":
		executeDB("UPDATE inv SET vat=" + strfloat2strint(r.Form.Get("vat")) + " WHERE id=" + r.Form.Get("id"))
	case "remove":
		executeDB("DELETE FROM inv_reg WHERE id=" + r.Form.Get("r_id"))
	case "Add":
		//id,qty
		s := strings.Split(r.Form.Get("p_id"), ",")
		p_id := s[0]
		avail, _ := strconv.Atoi(s[1])
		req, _ := strconv.Atoi(r.Form.Get("qty"))
		if avail >= req {
			executeDB("INSERT INTO inv_reg VALUES(" + r.Form.Get("r_id") + "," + r.Form.Get("id") + "," + p_id + "," + strfloat2strint(r.Form.Get("b_p")) + "," + strfloat2strint(r.Form.Get("s_p")) + "," + r.Form.Get("qty") + ")")
		} else {
			http.Redirect(w, r, "error", http.StatusSeeOther)
			return
		}
	case "Delete":
		executeDB("DELETE FROM inv_reg WHERE i_id=" + r.Form.Get("id"))
		deleteData("inv", r.Form.Get("id"))
		http.Redirect(w, r, "invoice", http.StatusSeeOther)
	}

	if r.Form.Get("id") == "" {
		http.Redirect(w, r, "invoice", http.StatusSeeOther)
		return
	}
	inv := get_invoice(r.Form.Get("id"))
	result := data{dbName, inv, getProductsInVan(strconv.FormatInt(inv.V_id, 10)), getNextID("inv_reg")}
	showFile(w, r, "editInvoice", result)
}

type vendor struct {
	Id        int64
	Name      string
	Phn       string
	Ad        string
	Dne       int64
	DeleteBTN bool
	Grns      []_grn
	Active    bool
}

func getVendor(id string) vendor {

	rows := getResultDB("SELECT * FROM ven WHERE id=" + id)
	tmp := vendor{}
	for rows.Next() {

		var id sql.NullInt64
		var name sql.NullString
		var phn sql.NullString
		var ad sql.NullString

		err := rows.Scan(&id, &name, &phn, &ad)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Name = name.String
		tmp.Phn = phn.String
		tmp.Ad = ad.String
	}
	rows.Close()
	return tmp
}

func getSimplyVendors(filter string, val string) [] vendor {
	tmp2 := []vendor{}

	rows := getResultDB("SELECT * FROM ven WHERE " + filter + "=" + val + "ORDER BY name")

	for rows.Next() {
		tmp := vendor{}

		var id sql.NullInt64
		var name sql.NullString
		var phn sql.NullString
		var ad sql.NullString

		err := rows.Scan(&id, &name, &phn, &ad)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Name = name.String
		tmp.Phn = phn.String
		tmp.Ad = ad.String

		tmp.DeleteBTN = true
		tmp.Dne = 0;
		for _, g := range tmp.Grns {
			tmp.Dne += g.Grnd_tot
		}
		if len(tmp.Grns) > 0 {
			tmp.DeleteBTN = false
		}
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func getVendors(filter string, val string) [] vendor {
	tmp2 := []vendor{}

	rows := getResultDB("SELECT * FROM ven WHERE " + filter + "=" + val + "ORDER BY name")

	i := true
	for rows.Next() {
		tmp := vendor{}
		tmp.Active = i
		i = false
		var id sql.NullInt64
		var name sql.NullString
		var phn sql.NullString
		var ad sql.NullString

		err := rows.Scan(&id, &name, &phn, &ad)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Name = name.String
		tmp.Phn = phn.String
		tmp.Ad = ad.String

		tmp.Grns = get_grns("v_id", strconv.FormatInt(id.Int64, 10))

		tmp.DeleteBTN = true
		tmp.Dne = 0;
		for _, g := range tmp.Grns {
			tmp.Dne += g.Grnd_tot
		}
		if len(tmp.Grns) > 0 {
			tmp.DeleteBTN = false
		}
		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func vendors(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.Form.Get("submit") == "Add" {
		insertData("ven", r.Form.Get("id") + ",'" + r.Form.Get("name") + "','" + r.Form.Get("phn") + "','" + r.Form.Get("ad") + "'")
	} else if r.Form.Get("submit") == "Save" {
		updateData("ven", r.Form.Get("id"), "name='" + r.Form.Get("name") + "', phn='" + r.Form.Get("phn") + "', ad='" + r.Form.Get("ad") + "'")
	} else if r.Form.Get("submit") == "Delete" {
		deleteData("ven", r.Form.Get("id"))
	}

	type sendData struct {
		Title   string
		Nid     int64
		Vendors []vendor
	}

	results := sendData{dbName, getNextID("ven"), getVendors("''", "''")}

	showFile(w, r, "vendors", results)
}

func delivery(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now()
	if r.Form.Get("submit") == "Add Vehicle" {
		executeDB("INSERT INTO van VALUES(" + r.Form.Get("id") + ",'" + r.Form.Get("des") + "')")
	} else if r.Form.Get("submit") == "Add Item" {
		//id,qty
		s := strings.Split(r.Form.Get("p_id"), ",")
		p_id := s[0]
		avail, _ := strconv.Atoi(s[1])
		req, _ := strconv.Atoi(r.Form.Get("qty"))
		if avail >= req {

			executeDB("INSERT INTO ldng VALUES(" + r.Form.Get("id") + "," + r.Form.Get("v_id") + "," + p_id + "," + r.Form.Get("qty") + ", '" + now.Format("2006-01-02") +
				"')")
		} else {
			http.Redirect(w, r, "error", http.StatusSeeOther)
			return
		}

	} else if r.Form.Get("submit") == "unload" {
		executeDB("INSERT INTO u_ldng VALUES(" + strconv.FormatInt(getNextID("u_ldng"), 10) + "," + r.Form.Get("v_id") + "," + r.Form.Get("p_id") + "," + r.Form.Get("qty") + ", '" + now.Format("2006-01-02") +
			"')")
		http.Redirect(w, r, "delivery", http.StatusSeeOther)
		return
	} else if r.Form.Get("submit") == "Delete" {
		deleteData("van", r.Form.Get("id"))
	} else if r.Form.Get("submit") == "Save" {
		updateData("van", r.Form.Get("id"), "des='" + r.Form.Get("des") + "'")
	}
	type sendData struct {
		Title   string
		Vans    []van
		Pro     []product
		NxtID   int64
		NxtLdID int64
	}

	results := sendData{dbName, getVansForDelivery("''", "''"), getProductsInMainStock("''", "''"), getNextID("van"), getNextID("ldng")}

	showFile(w, r, "delivery", results)

}

func load(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	type sendData struct {
		Title string
		Vans  []van
	}

	results := sendData{dbName, getVansForLoading("''", "''")}

	showFile(w, r, "load", results)
}

func unload(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	type sendData struct {
		Title string
		Vans  []van
	}

	results := sendData{dbName, getVansForUnoading("''", "''")}

	showFile(w, r, "unload", results)
}

type income struct {
	Date string
	Sale int64
	Tot  int64
}

func getIncome(from string, to string) ([] income, int64) {
	tmp2 := []income{}
	//rows := getResultDB("select sum(inv_reg.s_p*inv_reg.qty+inv.vat) as tot , inv.dte from inv_reg join inv on inv_reg.i_id=inv.id where inv.dte between '" + from + "' and '" + to + "' group by inv.dte order by inv.dte asc")
	rows := getResultDB("select sum(inv_reg.s_p*inv_reg.qty+inv.vat) as tot , inv.dte from inv_reg join inv on inv_reg.i_id=inv.id where inv.dte <= '" + to + "' group by inv.dte order by inv.dte asc")

	tot := int64(0)
	for rows.Next() {
		tmp := income{}

		var sale sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&sale, &dte)
		checkErr(err, errDBquery)

		tmp.Sale = sale.Int64
		tmp.Date = string(dte)
		tot += tmp.Sale
		tmp.Tot = tot

		invDte, err := time.Parse("2006-01-02", string(dte))

		if err != nil {
			checkErr(err, errTimeConversion)
		}
		frmDte, err := time.Parse("2006-01-02", from)

		if err != nil {
			checkErr(err, errTimeConversion)
		}

		diff := invDte.Sub(frmDte)

		if (diff.Hours() >= 0) {
			tmp2 = append(tmp2, tmp)
		}
	}
	rows.Close()
	return tmp2, tot
}

const (
	stdLongMonth = "January"
	stdMonth = "Jan"
	stdNumMonth = "1"
	stdZeroMonth = "01"
	stdLongWeekDay = "Monday"
	stdWeekDay = "Mon"
	stdDay = "2"
	stdUnderDay = "_2"
	stdZeroDay = "02"
	stdHour = "15"
	stdHour12 = "3"
	stdZeroHour12 = "03"
	stdMinute = "4"
	stdZeroMinute = "04"
	stdSecond = "5"
	stdZeroSecond = "05"
	stdLongYear = "2006"
	stdYear = "06"
	stdPM = "PM"
	stdpm = "pm"
	stdTZ = "MST"
	stdISO8601TZ = "Z0700"  // prints Z for UTC
	stdISO8601ColonTZ = "Z07:00" // prints Z for UTC
	stdNumTZ = "-0700"  // always numeric
	stdNumShortTZ = "-07"    // always numeric
	stdNumColonTZ = "-07:00" // always numeric
)

func stat(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	type data struct {
		Title   string
		Records []income
		Tot     int64
		From    string
		To      string
	}
	now := time.Now()

	now.Format("01/02/2006")

	//select sum(inv_reg.s_p*inv_reg.qty+inv.vat) as tot , inv.dte from inv_reg join inv on inv_reg.i_id=inv.id where inv.dte between '2017-09-07' and '2017-12-14' group by inv.dte order by inv.dte asc;
	from := now.Format("01") + "/01/" + now.Format("2006")
	to := now.Format("01/02/2006")

	if (r.Form.Get("from") != "") {
		from = r.Form.Get("from")
	}
	if (r.Form.Get("to") != "") {
		to = r.Form.Get("to")
	}

	//m,d,y
	s := strings.Split(from, "/")
	sqlFrom := s[2] + "-" + s[0] + "-" + s[1]

	s = strings.Split(to, "/")
	sqlTo := s[2] + "-" + s[0] + "-" + s[1]

	rcds, tot := getIncome(sqlFrom, sqlTo)
	if ( len(rcds) == 0) {
		rcds = append(rcds, income{now.Format("01/02/2006"), 0, tot})
	}
	results := data{dbName, rcds, tot, from, to}
	showFile(w, r, "stat", results)
}

type product struct {
	Id        int64
	Des       string
	S_p       int64
	B_p       int64
	Qty       int64
	QtyVan    int64
	QtyStk    int64
	DeleteBTN bool
}

func getProductsInMainStock(filter string, val string) [] product {
	tmp2 := []product{}

	rows := getResultDB("SELECT * FROM pro WHERE " + filter + "=" + val)

	for rows.Next() {
		tmp := product{}

		var id sql.NullInt64
		var des sql.NullString
		var s_p sql.NullInt64
		var b_p sql.NullInt64

		err := rows.Scan(&id, &des, &s_p, &b_p)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String
		tmp.S_p = s_p.Int64
		tmp.B_p = b_p.Int64

		invs := getInvoiceRecords("p_id", strconv.FormatInt(tmp.Id, 10))

		if len(invs) > 0 {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}

		//rows2 := getResultDB("select sum(qty)- (select sum(qty) from inv_reg where p_id=" + strconv.FormatInt(tmp.Id, 10) + ") as qty from grn_reg where p_id=" + strconv.FormatInt(tmp.Id, 10))


		var hav sql.NullInt64
		//var gne sql.NullInt64
		var ld sql.NullInt64
		var uld sql.NullInt64

		rows2 := getResultDB("select sum(qty) as qty from grn_reg where p_id=" + strconv.FormatInt(tmp.Id, 10))
		rows2.Next()
		err = rows2.Scan(&hav)
		if err != nil {
			hav.Int64 = 0
		}

		rows2.Close()

		//rows2 = getResultDB("select sum(qty) from inv_reg where p_id=" + strconv.FormatInt(tmp.Id, 10))
		//rows2.Next()
		//err = rows2.Scan(&gne)
		//if err != nil {
		//	gne.Int64 = 0
		//}

		//rows2.Close()

		rows2 = getResultDB("select  sum(qty) from u_ldng where p_id=" + strconv.FormatInt(tmp.Id, 10) + "  group by p_id")
		rows2.Next()
		err = rows2.Scan(&uld)
		if err != nil {
			uld.Int64 = 0
		}

		rows2.Close()

		rows2 = getResultDB("select  sum(qty) from ldng where p_id=" + strconv.FormatInt(tmp.Id, 10) + "  group by p_id")
		rows2.Next()
		err = rows2.Scan(&ld)
		if err != nil {
			ld.Int64 = 0
		}

		rows2.Close()

		//tmp.Qty = hav.Int64 - gne.Int64 - (ld.Int64 - uld.Int64)
		tmp.Qty = hav.Int64 - (ld.Int64 - uld.Int64)

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
}

func getProductsInVan(v_id string) [] productVan {
	tmp2 := []productVan{}

	rows2 := getResultDB("select p_id , sum(qty) as qty from ldng where v_id = " + v_id + "  group by p_id")
	for rows2.Next() {

		var p_id sql.NullInt64
		var qty sql.NullInt64

		err := rows2.Scan(&p_id, &qty)
		checkErr(err, errDBquery)

		tmp3 := getProductVan(strconv.FormatInt(p_id.Int64, 10))
		tmp3.Loaded = qty.Int64
		//select p_id , sum(qty) as qty from u_ldng where v_id = 0 and p_id = 0 group by p_id
		rows3 := getResultDB("select sum(qty) as qty from u_ldng where v_id = " + v_id + " and p_id = " + strconv.FormatInt(p_id.Int64, 10) + " group by p_id")
		rows3.Next()

		err = rows3.Scan(&qty)
		if err != nil {
			qty.Int64 = 0
		}

		tmp3.Unloaded = qty.Int64

		tmp3.Rest = tmp3.Loaded - tmp3.Unloaded
		rows3.Close()

		//to calculate sold items
		rows3 = getResultDB("select sum(qty) as qty from inv_reg LEFT JOIN inv ON inv_reg.i_id=inv.id where inv.v_id = " + v_id + " and inv_reg.p_id = " + strconv.FormatInt(p_id.Int64, 10) + " group by inv_reg.p_id")
		rows3.Next()

		err = rows3.Scan(&qty)
		if err != nil {
			qty.Int64 = 0
		}

		tmp3.Rest -= qty.Int64
		rows3.Close()

		tmp2 = append(tmp2, tmp3)
	}
	rows2.Close()

	return tmp2
}

func getProducts(filter string, val string) [] product {
	tmp2 := []product{}

	rows := getResultDB("SELECT * FROM pro WHERE " + filter + "=" + val)

	for rows.Next() {
		tmp := product{}

		var id sql.NullInt64
		var des sql.NullString
		var s_p sql.NullInt64
		var b_p sql.NullInt64

		err := rows.Scan(&id, &des, &s_p, &b_p)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.Des = des.String
		tmp.S_p = s_p.Int64
		tmp.B_p = b_p.Int64

		invs := getInvoiceRecords("p_id", strconv.FormatInt(tmp.Id, 10))
		grns := getGrnRecords("p_id", strconv.FormatInt(tmp.Id, 10))

		if (len(invs) > 0 || len(grns) > 0) {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}

		var hav sql.NullInt64
		var gne sql.NullInt64
		var ld sql.NullInt64
		var uld sql.NullInt64

		rows2 := getResultDB("select sum(qty) as qty from grn_reg where p_id=" + strconv.FormatInt(tmp.Id, 10))
		rows2.Next()
		err = rows2.Scan(&hav)
		if err != nil {
			hav.Int64 = 0
		}

		rows2.Close()

		rows2 = getResultDB("select sum(qty) from inv_reg where p_id=" + strconv.FormatInt(tmp.Id, 10))
		rows2.Next()
		err = rows2.Scan(&gne)
		if err != nil {
			gne.Int64 = 0
		}

		rows2.Close()

		rows2 = getResultDB("select  sum(qty) from u_ldng where p_id=" + strconv.FormatInt(tmp.Id, 10) + "  group by p_id")
		rows2.Next()
		err = rows2.Scan(&uld)
		if err != nil {
			uld.Int64 = 0
		}

		rows2.Close()

		rows2 = getResultDB("select  sum(qty) from ldng where p_id=" + strconv.FormatInt(tmp.Id, 10) + "  group by p_id")
		rows2.Next()
		err = rows2.Scan(&ld)
		if err != nil {
			ld.Int64 = 0
		}

		rows2.Close()

		tmp.Qty = hav.Int64 - gne.Int64
		tmp.QtyVan = ld.Int64 - uld.Int64 - gne.Int64
		tmp.QtyStk = tmp.Qty - tmp.QtyVan

		tmp2 = append(tmp2, tmp)
	}

	rows.Close()

	return tmp2
}

func products(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.Form.Get("submit") == "Add" {
		insertData("pro", r.Form.Get("id") + ",'" + r.Form.Get("des") + "'," + strfloat2strint(r.Form.Get("s_p")) + "," + strfloat2strint(r.Form.Get("b_p")))
	} else if r.Form.Get("submit") == "Save" {
		updateData("pro", r.Form.Get("id"), "des='" + r.Form.Get("des") + "', s_p=" + strfloat2strint(r.Form.Get("s_p")) + ", b_p=" + strfloat2strint(r.Form.Get("b_p")))
	} else if r.Form.Get("submit") == "Delete" {
		deleteData("pro", r.Form.Get("id"))
	}

	type sendData struct {
		Title    string
		Nid      int64
		Products []product
	}

	results := sendData{dbName, getNextID("pro"), getProducts("''", "''")}

	showFile(w, r, "products", results)
}

func getCustomerPaymentsForPayment(filter string, val string) []customerPayment {
	tmp := getCustomerPayments(filter, val)
	for i, p := range tmp {
		rows := getResultDB("select i_no , c_id from inv where id=" + strconv.FormatInt(p.I_id, 10))
		rows.Next()

		var i_no sql.NullString
		var c_id sql.NullInt64

		err := rows.Scan(&i_no, &c_id)
		checkErr(err, errDBquery)
		tmp[i].I_no = i_no.String

		rows.Close()

		rows = getResultDB("select name from cus where id=" + strconv.FormatInt(c_id.Int64, 10))
		rows.Next()

		var name sql.NullString
		err = rows.Scan(&name)
		checkErr(err, errDBquery)
		tmp[i].C_name = name.String

		rows.Close()
	}
	return tmp
}

func get_invoicesForPayments(filter string, val string) [] _invoice {
	tmp2 := []_invoice{}

	rows := getResultDB("SELECT * FROM inv WHERE " + filter + "=" + val + " ORDER BY dte DESC")

	for rows.Next() {
		tmp := _invoice{}

		var id sql.NullInt64
		var c_id sql.NullInt64
		var v_id sql.NullInt64
		var i_no sql.NullString
		var po_no sql.NullString
		var vat sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&id, &c_id, &v_id, &i_no, &po_no, &vat, &dte)
		checkErr(err, errDBquery)

		tmp.Id = id.Int64
		tmp.C_id = c_id.Int64
		tmp.V_id = v_id.Int64
		tmp.I_no = i_no.String
		tmp.Po_no = po_no.String
		tmp.Vat = vat.Int64
		tmp.Dte = string(dte)

		tmp.Records = getInvoiceRecords("i_id", strconv.FormatInt(id.Int64, 10))
		tmp.Payments = getCustomerPayments("i_id", strconv.FormatInt(id.Int64, 10))

		tmp.Cus = getSimplyCustomer(strconv.FormatInt(c_id.Int64, 10))

		tmp.PaymentsDone = 0
		for _, p := range tmp.Payments {
			tmp.PaymentsDone += p.Tot
		}

		tmp.Margine = 0.0
		tmp.Sub_tot = 0
		for _, p := range tmp.Records {
			tmp.Margine += p.Margine
			tmp.Sub_tot += p.Tot
		}

		tmp.Grnd_tot = tmp.Vat + tmp.Sub_tot

		tmp.RemainingPayment = tmp.Grnd_tot - tmp.PaymentsDone
		if tmp.Grnd_tot == 0 {
			tmp.Progress = 100
		} else {
			tmp.Progress = (tmp.PaymentsDone * 100) / tmp.Grnd_tot
		}

		if len(tmp.Records) > 0 {
			tmp.DeleteBTN = false
		} else {
			tmp.DeleteBTN = true
		}
		if tmp.RemainingPayment > 0 {
			tmp2 = append(tmp2, tmp)
		}
	}
	rows.Close()
	return tmp2
}

func payment(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.Form.Get("submit") == "Add" {
		//id,remaining
		s := strings.Split(r.Form.Get("i_id"), ",")
		i_id := s[0]
		avail, _ := strconv.Atoi(s[1])
		req, _ := strconv.Atoi(r.Form.Get("tot"))
		if avail >= req {

			insertData("cus_pay", r.Form.Get("id") + ",'" + r.Form.Get("dte") + "'," + i_id + ",'" + r.Form.Get("des") + "'," + strfloat2strint(r.Form.Get("tot")))
		} else {
			http.Redirect(w, r, "error", http.StatusSeeOther)
			return
		}
	} else if r.Form.Get("submit") == "Save" {
		updateData("cus_pay", r.Form.Get("id"), "des='" + r.Form.Get("des") + "', dte='" + r.Form.Get("dte") + "', tot=" + strfloat2strint(r.Form.Get("tot")))
	} else if r.Form.Get("submit") == "Delete" {
		deleteData("cus_pay", r.Form.Get("id"))
	}

	type sendData struct {
		Title    string
		Nid      int64
		Dte      string
		Payments []customerPayment
		Invoices []_invoice
	}
	now := time.Now()

	var results sendData

	if r.Form.Get("q") == "" {
		results = sendData{dbName, getNextID("cus_pay"), now.Format("01/02/2006"), getCustomerPaymentsForPayment("''", "''"), get_invoicesForPayments("''", "''")}
	} else {
		results = sendData{dbName, getNextID("cus_pay"), now.Format("01/02/2006"), getCustomerPaymentsForPayment("i_id", r.Form.Get("q")), get_invoicesForPayments("id", r.Form.Get("q"))}
	}

	showFile(w, r, "payment", results)
}

func home(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.Form.Get("submit") != "" {
		dbName = r.Form.Get("submit")
		initDatabase()
		http.Redirect(w, r, "stat", http.StatusSeeOther)
		return
	}

	showFile(w, r, "home", "")
}

func startService() {
	//err := http.ListenAndServeTLS(":8080", "hostcert.pem", "hostkey.pem", nil)
	//err := http.ListenAndServe("localhost:8008", nil)
	//if err != nil {
	//	log.Fatal("ListenAndServe: ", err)
	//}

	s := &http.Server{
		Addr:           ":8008",
		Handler:        nil,
		ReadTimeout:    2 * time.Second,
		WriteTimeout:   2 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())

}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		checkErr(err,errBrowser)
	}
}

func int2floatStr(in int64) string {
	val := in / 100
	diff := in - (val * 100)
	if (diff < 10) {
		return strconv.FormatInt(val, 10) + ".0" + strconv.FormatInt(diff, 10)
	} else {
		return strconv.FormatInt(val, 10) + "." + strconv.FormatInt(diff, 10)
	}
}

func strfloat2strint(in string) string {
	s := strings.Split(in, ".")
	val := s[0]
	if (len(s) == 1) {
		val += "00"
	} else if (len(s) == 2) {
		ss := strings.Split(s[1], "")

		if (len(ss) == 1) {
			val += ss[0] + "0"
		} else if (len(ss) > 1) {
			val += ss[0] + ss[1]
		} else {
			val += "00"
		}
	}
	return val
}

func main() {
	initFiles()
	initDatabase()
	http.HandleFunc("/invoice", invoice)
	http.HandleFunc("/grn", grn)
	http.HandleFunc("/customers", customers)
	http.HandleFunc("/vendors", vendors)
	http.HandleFunc("/delivery", delivery)
	http.HandleFunc("/load", load)
	http.HandleFunc("/unload", unload)
	http.HandleFunc("/stat", stat)
	http.HandleFunc("/products", products)
	http.HandleFunc("/payment", payment)
	http.HandleFunc("/editGRN", editGRN)
	http.HandleFunc("/editInvoice", editInvoice)
	http.HandleFunc("/home", home)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if (r.URL.Path == "/") {
			http.Redirect(w, r, "home", http.StatusSeeOther)
		} else {
			http.ServeFile(w, r, "helper/" + r.URL.Path[1:])
		}
	})
	openbrowser("http://localhost:8008")
	startService()
}

func initFiles() {
	files = make(map[string]string)
	files["customers"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <link href="css/plugins/dataTables/dataTables.bootstrap.css" rel="stylesheet">
    <link href="css/plugins/dataTables/dataTables.responsive.css" rel="stylesheet">
    <link href="css/plugins/dataTables/dataTables.tableTools.min.css" rel="stylesheet">

</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="active">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                <div class="navbar-header">
                    <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i class="fa fa-bars"></i>
                    </a>
                </div>
                <div class="">
                    <a data-toggle="modal" href="#Addnew" class="btn btn-primary minimalize-styl-2">Add new</a>
                </div>

            </nav>
        </div>


        <div id="Addnew" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form role="form" action="/customers" method="POST">
                                    <div class="form-group"><label>ID</label>
                                        <input name="id" readonly required type="text" placeholder="ID"
                                               class="form-control" value="{{.Nid}}">
                                    </div>
                                    <div class="form-group"><label>Name</label>
                                        <input name="name" required type="text" placeholder="Name"
                                               class="form-control" value="">
                                    </div>
                                    <div class="form-group"><label>Phone</label>
                                        <input name="phn" type="text" placeholder="Phone"
                                               class="form-control" value="">
                                    </div>
                                    <div class="form-group"><label>Address</label>
                                        <input name="ad" type="text" placeholder="Address"
                                               class="form-control" value="">
                                    </div>

                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs"
                                                       value="Add"></strong>
                                        <strong><input type="reset" class="btn btn-sm btn-warning pull-right m-t-n-xs"
                                                       value="Reset"></strong>
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        {{range .Customers}}
        <div id="c{{.Id}}" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form role="form" action="/customers" method="POST">
                                    <div class="form-group"><label>ID</label>
                                        <input name="id" readonly required type="text" placeholder="ID"
                                               class="form-control" value="{{.Id}}">
                                    </div>
                                    <div class="form-group"><label>Name</label>
                                        <input name="name" required type="text" placeholder="Name"
                                               class="form-control" value="{{.Name}}">
                                    </div>
                                    <div class="form-group"><label>Phone</label>
                                        <input name="phn" type="text" placeholder="Phone"
                                               class="form-control" value="{{.Phn}}">
                                    </div>
                                    <div class="form-group"><label>Address</label>
                                        <input name="ad" type="text" placeholder="Address"
                                               class="form-control" value="{{.Ad}}">
                                    </div>

                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs" value="Save"></strong>
                                        {{if .DeleteBTN}}
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-danger pull-right m-t-n-xs" value="Delete"></strong>
                                        {{end}}
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        {{end}}


        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-7">
                    <div class="ibox">
                        <div class="ibox-content">
                            <div class="clients-list">
                                <div class="tab-content">
                                    <div class="tab-pane active">
                                        <div class="full-height-scroll">
                                            <div class="table-responsive">
                                                <table class="table table-striped table-hover dataTables-example">
                                                    <thead>
                                                    <tr>
                                                        <th>Name</th>
                                                        <th>Phone</th>
                                                        <th>Pending Dues</th>
                                                        <th>Due Progress</th>
                                                        <th>Option</th>
                                                    </tr>
                                                    </thead>
                                                    <tbody>
                                                    {{range .Customers}}
                                                    <tr>
                                                        <td><a data-toggle="tab" href="#cus{{.Id}}" class="client-link">
                                                            {{.Name}}
                                                        </a></td>
                                                        <td>{{.Phn}}</td>
                                                        <td>{{dec .Due}}</td>

                                                        <td>
                                                            {{.Pro}}%
                                                        </td>


                                                        <td><a data-toggle="modal"
                                                               class="btn btn-info btn-sm btn-outline" href="#c{{.Id}}">View
                                                            /
                                                            Edit</a>
                                                        </td>
                                                    </tr>
                                                    {{end}}
                                                    </tbody>
                                                </table>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-sm-5">
                    <div class="ibox ">

                        <div class="ibox-content">
                            <div class="tab-content">
                                {{range .Customers}}
                                <div id="cus{{.Id}}" class="tab-pane {{if .Active}} active {{end}}">
                                    <div class="m-b-lg">
                                        <h2>{{.Name}}</h2>
                                        <br>
                                        <p>
                                            {{.Phn}}
                                            <br>
                                            {{.Ad}}
                                        </p>
                                        <div>
                                            payments done {{dec .Dne}} / payments pending {{dec .Due}}
                                            <br>
                                            <small>
                                                {{.Pro}}%
                                            </small>
                                            <div class="progress progress-mini">
                                                <div style="width: {{.Pro}}%;" class="progress-bar"></div>
                                            </div>
                                        </div>
                                    </div>
                                    <div class="client-detail">
                                        <div class="full-height-scroll">
                                            <strong>Timeline activity</strong>

                                            <div id="vertical-timeline" class="vertical-container dark-timeline">
                                                {{range .Invoices}}
                                                <div class="vertical-timeline-block">
                                                    <div class="vertical-timeline-icon navy-bg">
                                                        <i class="fa fa-usd"></i>
                                                    </div>
                                                    <div class="vertical-timeline-content">

                                                        <p>Inv. No.:{{.I_no}} P.O. No.:{{.Po_no}}
                                                            <br>
                                                            Margine: {{.Margine}}
                                                            <br>
                                                            Total:{{dec .Grnd_tot}}
                                                            <button type="button" class="btn btn-info btn-xs"
                                                                    data-toggle="modal" data-target="#i{{.Id}}">
                                                                invoice
                                                            </button>
                                                            <button type="button" class="btn btn-success btn-xs"
                                                                    data-toggle="modal" data-target="#p{{.Id}}">
                                                                payments
                                                            </button>
                                                        </p>
                                                        <br>
                                                        Payment Completed {{.Progress}}%
                                                        <div class="progress progress-mini">
                                                            <div style="width: {{.Progress}}%;"
                                                                 class="progress-bar"></div>
                                                        </div>
                                                        <span class="vertical-date small text-muted"> Date: {{.Dte}} </span>
                                                    </div>
                                                </div>

                                                {{end}}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                {{end}}

                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
`+footer+`
    </div>
</div>

{{range .Customers}}
{{range .Invoices}}
<div class="modal inmodal fade" id="i{{.Id}}" tabindex="-1" role="dialog" aria-hidden="true">
    <div class="modal-dialog modal-md">
        <div class="modal-content">
            <div class="modal-header">
                <button type="button" class="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span
                        class="sr-only">Close</span></button>
                <h4 class="modal-title">Inv. No.:{{.I_no}} P.O. No.:{{.Po_no}}</h4>
                <h5>Date:{{.Dte}}</h5>
            </div>
            <div class="modal-body">
                <table class="table table-striped table-bordered">
                    <thead>
                    <tr>
                        <th>Description</th>
                        <th>Qty</th>
                        <th>U.Price</th>
                        <th>Total</th>
                    </tr>
                    </thead>
                    <tbody>
                    {{range .Records}}
                    <tr>
                        <td>{{.P_des}}</td>
                        <td>{{.Qty}}</td>
                        <td>{{dec .S_p}}</td>
                        <td>{{dec .Tot}}</td>
                    </tr>
                    {{end}}
                    <tr>
                        <td></td>
                        <td></td>
                        <td></td>
                        <td></td>
                    </tr>
                    <tr>
                        <td><strong>Sub Total</strong></td>
                        <td></td>
                        <td></td>
                        <td>{{dec .Sub_tot}}</td>
                    </tr>
                    <tr>
                        <td><strong>Vat</strong></td>
                        <td></td>
                        <td></td>
                        <td>{{dec .Vat}}</td>
                    </tr>
                    <tr>
                        <td><strong>Grand Total</strong></td>
                        <td></td>
                        <td></td>
                        <td>{{dec .Grnd_tot}}</td>
                    </tr>
                    </tbody>
                </table>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-white btn-sm" data-dismiss="modal">Close</button>
                <a href="editInvoice?id={{.Id}}" type="button" class="btn btn-warning btn-sm">Edit/View</a>
            </div>
        </div>
    </div>
</div>

<div class="modal inmodal fade" id="p{{.Id}}" tabindex="-1" role="dialog" aria-hidden="true">
    <div class="modal-dialog modal-md">
        <div class="modal-content">
            <div class="modal-header">
                <button type="button" class="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span
                        class="sr-only">Close</span></button>
                <h4 class="modal-title">Inv. No.:{{.I_no}} P.O. No.:{{.Po_no}}</h4>
                <h5>Date:{{.Dte}}</h5>
            </div>
            <div class="modal-body">
                <table class="table table-striped table-bordered">
                    <thead>
                    <tr>
                        <th>Date</th>
                        <th>Amount</th>
                    </tr>
                    </thead>
                    <tbody>
                    {{range .Payments}}
                    <tr>
                        <td>{{.Dte}}</td>
                        <td>{{dec .Tot}}</td>
                    </tr>
                    {{end}}
                    <tr>
                        <td></td>
                        <td></td>
                    </tr>
                    <tr>
                        <td><strong>Total</strong></td>
                        <td>{{dec .Grnd_tot}}</td>
                    </tr>
                    <tr>
                        <td><strong>Payments Done</strong></td>
                        <td>{{dec .PaymentsDone}}</td>
                    </tr>
                    <tr>
                        <td><strong>Due</strong></td>
                        <td>{{dec .RemainingPayment}}</td>
                    </tr>
                    </tbody>
                </table>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-white btn-sm" data-dismiss="modal">Close</button>
                <a href="payment?q={{.Id}}" type="button" class="btn btn-warning btn-sm">Edit/View</a>
            </div>
        </div>
    </div>
</div>
{{end}}
{{end}}
<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>


<script src="js/plugins/dataTables/jquery.dataTables.js"></script>
<script src="js/plugins/dataTables/dataTables.bootstrap.js"></script>
<script src="js/plugins/dataTables/dataTables.responsive.js"></script>
<script src="js/plugins/dataTables/dataTables.tableTools.min.js"></script>

<script>
        $(document).ready(function() {
            $('.dataTables-example').DataTable({

            });


        });


</script>
</body>
</html>

	`
	files["db.sql"] = `
	CREATE TABLE IF NOT EXISTS cus (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    phn VARCHAR(255),
    ad VARCHAR(255)
);


CREATE TABLE IF NOT EXISTS cus_pay (
    id INT PRIMARY KEY,
     dte DATE,
     i_id INT,
     des VARCHAR(255),
     tot INT
);

CREATE TABLE IF NOT EXISTS ven (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    phn VARCHAR(255),
    ad VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS pro (
    id INT PRIMARY KEY,
    des VARCHAR(255),
    s_p INT,
    b_p INT
);

CREATE TABLE IF NOT EXISTS van (
    id INT PRIMARY KEY,
    des VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS ldng (
id INT PRIMARY KEY,
    v_id INT,
    p_id INT,
    qty INT,
    dte DATE
);


CREATE TABLE IF NOT EXISTS u_ldng (
id INT PRIMARY KEY,
    v_id INT,
    p_id INT,
    qty INT,
    dte DATE
);

CREATE TABLE IF NOT EXISTS grn (
    id INT PRIMARY KEY,
    v_id INT,
    g_no VARCHAR(255),
    vat INT,
    dte DATE
);


CREATE TABLE IF NOT EXISTS grn_reg (
    id INT PRIMARY KEY,
    g_id INT,
    p_id INT,
    b_p INT,
    qty INT
);

CREATE TABLE IF NOT EXISTS inv (
    id INT PRIMARY KEY,
    c_id INT,
    v_id INT,
    i_no VARCHAR(255),
    po_no VARCHAR(255),
    vat INT,
    dte DATE
);


CREATE TABLE IF NOT EXISTS inv_reg (
    id INT PRIMARY KEY,
    i_id INT,
    p_id INT,
    b_p INT,
    s_p INT,
    qty INT
);

	`
	files["delivery"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- Sweet Alert -->
    <link href="css/plugins/sweetalert/sweetalert.css" rel="stylesheet">

</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="active">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    {{range .Vans}}
    <div id="editvan_{{.Id}}" class="modal fade" aria-hidden="true">
        <div class="modal-dialog">
            <div class="modal-content">
                <div class="modal-body">
                    <div class="row">
                        <div class=""><h3 class="m-t-none m-b">Details</h3>

                            <form role="form" action="/delivery" method="POST">
                                <div class="form-group"><label>ID</label>
                                    <input name="id" readonly required type="text" placeholder="ID"
                                           class="form-control" value="{{.Id}}">
                                </div>
                                <div class="form-group"><label>Description</label>
                                    <input name="des" required type="text" placeholder="Description"
                                           class="form-control" value="{{.Des}}">
                                </div>
                                <div>
                                    <strong><input name="submit" type="submit"
                                                   class="btn btn-sm btn-primary pull-right m-t-n-xs"
                                                   value="Save"></strong>
                                    {{if .Delete}}
                                    <strong><input name="submit" type="submit"
                                                   class="btn btn-sm btn-danger pull-right m-t-n-xs"
                                                   value="Delete"></strong>
                                    {{end}}
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    {{end}}


    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <form role="form" action="/delivery" method="POST">
                <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                    <div class="navbar-header">
                        <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i
                                class="fa fa-bars"></i>
                        </a>
                    </div>

                    <div class="minimalize-styl-2">
                        <input hidden value="{{.NxtID}}" name="id">
                        <input name="des" required type="text" placeholder="Description" value="">
                    </div>
                    <div class="minimalize-styl-2">
                        <input type="submit" name="submit" class="btn btn-primary " value="Add Vehicle">
                    </div>
                </nav>
            </form>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-12">
                    <div class="ibox">
                        <div class="ibox-content">
                            <h2>Vehicles</h2>
                            <div class="input-group">
                                <input type="text" class="input form-control"
                                       id="filter"
                                       placeholder="Search in Vehicles">
                                </span>
                            </div>
                            <div class="clients-list">
                                <ul class="nav nav-tabs">
                                    {{range .Vans}}
                                    <li {{if .Active}}class="active" {{end}}><a data-toggle="tab" href="#van{{.Id}}"><i
                                            class="fa fa-bus"></i>
                                        {{.Des}}</a>
                                    </li>
                                    {{end}}
                                </ul>
                                <div class="tab-content">
                                    {{$pro := .Pro}}
                                    {{$newid := .NxtLdID}}
                                    {{range .Vans}}
                                    {{$v := .Id}}
                                    <div id="van{{.Id}}" class="tab-pane {{if .Active}}active{{end}}">
                                        <a data-toggle="modal" class="btn btn-xs btn-warning btn-outline"
                                           href="#editvan_{{.Id}}">edit</a>
                                        <br>
                                        <form role="form" action="/delivery" method="POST"
                                              onsubmit="return validateForm()" name="myForm">
                                            <select name="p_id" data-placeholder="Choose an Item" class="chosen-select "
                                                    style="width:550px;">
                                                {{range $pro}}
                                                {{if .Qty}}
                                                <option value="{{.Id}},{{.Qty}}">{{.Des}} ________available: {{.Qty}}
                                                </option>
                                                {{end}}
                                                {{end}}
                                            </select>
                                            <input hidden value="{{$newid}}" name="id">
                                            <input hidden value="{{.Id}}" name="v_id">
                                            <input name="qty" required type="number" placeholder="Qty." value="">
                                            <input type="submit" class="btn btn-sm btn-primary" name="submit"
                                                   value="Add Item">
                                        </form>
                                        <div class="full-height-scroll">
                                            <div class="col-lg-12">
                                                <div class="ibox float-e-margins">
                                                    <div class="ibox-content">
                                                        <table class="footable table table-stripped" data-page-size="8"
                                                               data-filter=#filter>
                                                            <thead>
                                                            <tr>
                                                                <th>Product</th>
                                                                <th>Van Stock</th>
                                                            </tr>
                                                            </thead>
                                                            <tbody>
                                                            {{range .Pro}}
                                                            {{if .Rest}}
                                                            <tr class="gradeU">
                                                                <td>{{.Des}}</td>
                                                                <td>{{.Rest}}
                                                                    <a class="btn btn-xs btn-danger pull-right"
                                                                       href="delivery?v_id={{$v}}&p_id={{.Id}}&submit=unload&qty={{.Rest}}">Unload</a>
                                                                </td>
                                                            </tr>
                                                            {{end}}
                                                            {{end}}
                                                            </tbody>
                                                            <tfoot>
                                                            <tr>
                                                                <td colspan="5">
                                                                    <ul class="pagination pull-right"></ul>
                                                                </td>
                                                            </tr>
                                                            </tfoot>
                                                        </table>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>


                                    </div>
                                    {{end}}
                                </div>

                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
`+footer+`
    </div>
</div>
<script>
function validateForm() {
    var x = document.forms["myForm"]["p_id"].value;
    var qty = document.forms["myForm"]["qty"].value;
    var splits = x.split(',', 2);
    if (parseInt(splits[1])<parseInt(qty)) {
     swal("Invalied Qty.", "Quantity must be lessthan or equal to "+splits[1], "warning");
        return false;
    }
}
</script>
<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>


<!-- FooTable -->
<script src="js/plugins/footable/footable.all.min.js"></script>

<!-- Sweet alert -->
<script src="js/plugins/sweetalert/sweetalert.min.js"></script>

<!-- Page-Level Scripts -->
<script>
        $(document).ready(function() {

            $('.footable').footable();
            $('.footable2').footable();

        });

</script>

</body>
</html>

	`
	files["editGRN"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">


    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <link href="css/plugins/iCheck/custom.css" rel="stylesheet">

    <link href="css/plugins/chosen/chosen.css" rel="stylesheet">

    <link href="css/plugins/colorpicker/bootstrap-colorpicker.min.css" rel="stylesheet">

    <link href="css/plugins/cropper/cropper.min.css" rel="stylesheet">

    <link href="css/plugins/switchery/switchery.css" rel="stylesheet">

    <link href="css/plugins/jasny/jasny-bootstrap.min.css" rel="stylesheet">

    <link href="css/plugins/nouslider/jquery.nouislider.css" rel="stylesheet">

    <link href="css/plugins/datapicker/datepicker3.css" rel="stylesheet">

    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.css" rel="stylesheet">
    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.skinFlat.css" rel="stylesheet">

    <link href="css/plugins/awesome-bootstrap-checkbox/awesome-bootstrap-checkbox.css" rel="stylesheet">

    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/select2/select2.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="active">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>
    {{$g := .Grn}}
    {{$p := .Products}}
    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <form role="form" action="/editGRN" method="POST" onsubmit="return validateForm()" name="myForm">
                <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                    <div class="navbar-header">
                        <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i
                                class="fa fa-bars"></i>
                        </a>
                    </div>
                    <div class="minimalize-styl-2 input-group">
                        <select name="p_id" data-placeholder="Choose an Item" class="chosen-select "
                                style="width:350px;">
                            {{range .Products}}
                            <option value="{{.Id}},{{dec .B_p}}">{{.Des}} -----Buy:{{dec .B_p}}</option>
                            {{end}}
                        </select>
                    </div>

                    <div class="minimalize-styl-2">
                        <input hidden value="{{$g.Id}}" name="id">
                        <input hidden value="{{.NxtID}}" name="r_id">
                        <input name="b_p" type="number" placeholder="Buy Price" value="" step="0.01">
                        <input name="qty" required type="number" placeholder="Qty" value="">
                        <input type="submit" name="submit" class="btn btn-primary " value="Add">
                    </div>
                </nav>
            </form>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-lg-12">
                    <div class="wrapper wrapper-content animated fadeInRight">
                        <div class="ibox-content p-xl">
                            <form action="/editGRN" method="POST">
                                <div class="row">
                                    <div class="col-sm-6">
                                        <h4>GRN No.</h4>
                                        <h4 class="text-navy">{{$g.G_no}}</h4>
                                        <span>To:</span>
                                        <address>
                                            <strong>{{($g.Ven).Name}}</strong><br>
                                            {{($g.Ven).Ad}}
                                            <br>
                                            {{($g.Ven).Phn}}
                                        </address>
                                        <p>
                                            <span><strong>GRN Date:</strong> {{$g.Dte}}</span>
                                        </p>
                                    </div>
                                </div>

                                <div class="table-responsive m-t">
                                    <table class="table invoice-table">
                                        <thead>
                                        <tr>
                                            <th>Item List</th>
                                            <th>Quantity</th>
                                            <th>Unit Price</th>
                                            <th>Total Price</th>
                                        </tr>
                                        </thead>
                                        <tbody>

                                        {{range $g.Records}}
                                        <tr>
                                            <td>
                                                <strong>{{.P_des}}</strong>

                                                <div class="pull-right">
                                                    <a href="editGRN?id={{$g.Id}}&r_id={{.Id}}&submit=remove"
                                                       class="btn btn-sm btn-danger">remove</a>
                                                </div>
                                            </td>
                                            <td>{{.Qty}}</td>
                                            <td>{{dec .B_p}}</td>
                                            <td>{{dec .Tot}}</td>
                                        </tr>
                                        {{end}}

                                        </tbody>
                                    </table>
                                </div><!-- /table-responsive -->

                                <table class="table invoice-total">
                                    <tbody>
                                    <tr>
                                        <td><strong>Sub Total :</strong></td>
                                        <td>{{dec $g.Sub_tot}}</td>
                                    </tr>
                                    <tr>
                                        <td><strong>Vat :</strong></td>
                                        <td><input name="vat" type="number" required value="{{dec $g.Vat}}" step="0.01"><input hidden
                                                                                                               name="id"
                                                                                                               value="{{$g.Id}}">
                                        </td>
                                    </tr>
                                    <tr>
                                        <td><strong>Grand Total :</strong></td>
                                        <td>{{dec $g.Grnd_tot}}</td>
                                    </tr>
                                    </tbody>
                                </table>
                                <div class="col-sm-12">
                                    <button type="submit" name="submit" value="Save" class="btn btn-primary pull-right">
                                        Save
                                    </button>
                                    <button type="submit" name="submit" value="Delete" class="btn btn-danger">Delete GRN
                                    </button>
                                </div>
                                <br>
                                <input hidden value="{{$g.Id}}" name="id">

                                <div class="input-group">
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>
`+footer+`
    </div>
</div>

<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Chosen -->
<script src="js/plugins/chosen/chosen.jquery.js"></script>

<!-- JSKnob -->
<script src="js/plugins/jsKnob/jquery.knob.js"></script>

<!-- Input Mask-->
<script src="js/plugins/jasny/jasny-bootstrap.min.js"></script>

<!-- Data picker -->
<script src="js/plugins/datapicker/bootstrap-datepicker.js"></script>

<!-- NouSlider -->
<script src="js/plugins/nouslider/jquery.nouislider.min.js"></script>

<!-- Switchery -->
<script src="js/plugins/switchery/switchery.js"></script>

<!-- IonRangeSlider -->
<script src="js/plugins/ionRangeSlider/ion.rangeSlider.min.js"></script>

<!-- iCheck -->
<script src="js/plugins/iCheck/icheck.min.js"></script>

<!-- MENU -->
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>

<!-- Color picker -->
<script src="js/plugins/colorpicker/bootstrap-colorpicker.min.js"></script>

<!-- Clock picker -->
<script src="js/plugins/clockpicker/clockpicker.js"></script>

<!-- Image cropper -->
<script src="js/plugins/cropper/cropper.min.js"></script>

<!-- Date range use moment.js same as full calendar plugin -->
<script src="js/plugins/fullcalendar/moment.min.js"></script>

<!-- Date range picker -->
<script src="js/plugins/daterangepicker/daterangepicker.js"></script>

<!-- Select2 -->
<script src="js/plugins/select2/select2.full.min.js"></script>
<script>
        $(document).ready(function(){

            var $image = $(".image-crop > img")
            $($image).cropper({
                aspectRatio: 1.618,
                preview: ".img-preview",
                done: function(data) {
                    // Output the result data for cropping image.
                }
            });

            var $inputImage = $("#inputImage");
            if (window.FileReader) {
                $inputImage.change(function() {
                    var fileReader = new FileReader(),
                            files = this.files,
                            file;

                    if (!files.length) {
                        return;
                    }

                    file = files[0];

                    if (/^image\/\w+$/.test(file.type)) {
                        fileReader.readAsDataURL(file);
                        fileReader.onload = function () {
                            $inputImage.val("");
                            $image.cropper("reset", true).cropper("replace", this.result);
                        };
                    } else {
                        showMessage("Please choose an image file.");
                    }
                });
            } else {
                $inputImage.addClass("hide");
            }

            $("#download").click(function() {
                window.open($image.cropper("getDataURL"));
            });

            $("#zoomIn").click(function() {
                $image.cropper("zoom", 0.1);
            });

            $("#zoomOut").click(function() {
                $image.cropper("zoom", -0.1);
            });

            $("#rotateLeft").click(function() {
                $image.cropper("rotate", 45);
            });

            $("#rotateRight").click(function() {
                $image.cropper("rotate", -45);
            });

            $("#setDrag").click(function() {
                $image.cropper("setDragMode", "crop");
            });

            $('#data_1 .input-group.date').datepicker({
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                calendarWeeks: true,
                autoclose: true
            });

            $('#data_2 .input-group.date').datepicker({
                startView: 1,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                format: "dd/mm/yyyy"
            });

            $('#data_3 .input-group.date').datepicker({
                startView: 2,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            $('#data_4 .input-group.date').datepicker({
                minViewMode: 1,
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                todayHighlight: true
            });

            $('#data_5 .input-daterange').datepicker({
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            var elem = document.querySelector('.js-switch');
            var switchery = new Switchery(elem, { color: '#1AB394' });

            var elem_2 = document.querySelector('.js-switch_2');
            var switchery_2 = new Switchery(elem_2, { color: '#ED5565' });

            var elem_3 = document.querySelector('.js-switch_3');
            var switchery_3 = new Switchery(elem_3, { color: '#1AB394' });

            $('.i-checks').iCheck({
                checkboxClass: 'icheckbox_square-green',
                radioClass: 'iradio_square-green'
            });

            $('.demo1').colorpicker();

            var divStyle = $('.back-change')[0].style;
            $('#demo_apidemo').colorpicker({
                color: divStyle.backgroundColor
            }).on('changeColor', function(ev) {
                        divStyle.backgroundColor = ev.color.toHex();
                    });

            $('.clockpicker').clockpicker();

            $('input[name="daterange"]').daterangepicker();

            $('#reportrange span').html(moment().subtract(29, 'days').format('MMMM D, YYYY') + ' - ' + moment().format('MMMM D, YYYY'));

            $('#reportrange').daterangepicker({
                format: 'MM/DD/YYYY',
                startDate: moment().subtract(29, 'days'),
                endDate: moment(),
                minDate: '01/01/2012',
                maxDate: '12/31/2015',
                dateLimit: { days: 60 },
                showDropdowns: true,
                showWeekNumbers: true,
                timePicker: false,
                timePickerIncrement: 1,
                timePicker12Hour: true,
                ranges: {
                    'Today': [moment(), moment()],
                    'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
                    'Last 7 Days': [moment().subtract(6, 'days'), moment()],
                    'Last 30 Days': [moment().subtract(29, 'days'), moment()],
                    'This Month': [moment().startOf('month'), moment().endOf('month')],
                    'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
                },
                opens: 'right',
                drops: 'down',
                buttonClasses: ['btn', 'btn-sm'],
                applyClass: 'btn-primary',
                cancelClass: 'btn-default',
                separator: ' to ',
                locale: {
                    applyLabel: 'Submit',
                    cancelLabel: 'Cancel',
                    fromLabel: 'From',
                    toLabel: 'To',
                    customRangeLabel: 'Custom',
                    daysOfWeek: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr','Sa'],
                    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
                    firstDay: 1
                }
            }, function(start, end, label) {
                console.log(start.toISOString(), end.toISOString(), label);
                $('#reportrange span').html(start.format('MMMM D, YYYY') + ' - ' + end.format('MMMM D, YYYY'));
            });

            $(".select2_demo_1").select2();
            $(".select2_demo_2").select2();
            $(".select2_demo_3").select2({
                placeholder: "Select a state",
                allowClear: true
            });


        });
        var config = {
                '.chosen-select'           : {},
                '.chosen-select-deselect'  : {allow_single_deselect:true},
                '.chosen-select-no-single' : {disable_search_threshold:10},
                '.chosen-select-no-results': {no_results_text:'Oops, nothing found!'},
                '.chosen-select-width'     : {width:"95%"}
                }
            for (var selector in config) {
                $(selector).chosen(config[selector]);
            }

        $("#ionrange_1").ionRangeSlider({
            min: 0,
            max: 5000,
            type: 'double',
            prefix: "$",
            maxPostfix: "+",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_2").ionRangeSlider({
            min: 0,
            max: 10,
            type: 'single',
            step: 0.1,
            postfix: " carats",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_3").ionRangeSlider({
            min: -50,
            max: 50,
            from: 0,
            postfix: "°",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_4").ionRangeSlider({
            values: [
                "January", "February", "March",
                "April", "May", "June",
                "July", "August", "September",
                "October", "November", "December"
            ],
            type: 'single',
            hasGrid: true
        });

        $("#ionrange_5").ionRangeSlider({
            min: 10000,
            max: 100000,
            step: 100,
            postfix: " km",
            from: 55000,
            hideMinMax: true,
            hideFromTo: false
        });

        $(".dial").knob();

        $("#basic_slider").noUiSlider({
            start: 40,
            behaviour: 'tap',
            connect: 'upper',
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#range_slider").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#drag-fixed").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag-fixed',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });


</script>

 <!-- Jquery Validate -->
    <script src="js/plugins/validate/jquery.validate.min.js"></script>

    <script>
         $(document).ready(function(){

             $(".cform").validate({
                 rules: {
                     password: {
                         required: true,
                         minlength: 3
                     },
                     url: {
                         required: true,
                         url: true
                     },
                     number: {
                         required: true,
                         number: true
                     },
                     min: {
                         required: true,
                         minlength: 6
                     },
                     max: {
                         required: true,
                         maxlength: 4
                     }
                 }
             });
        });
    </script>

<script>
function validateForm() {
    var x = document.forms["myForm"]["p_id"].value;
    var b = document.forms["myForm"]["b_p"].value;
    var splits = x.split(',', 2);

    if(b==''){
    document.forms["myForm"]["b_p"].value=splits[1]
    }
}
</script>
</body>
</html>

	`
	files["editInvoice"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">


    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <link href="css/plugins/iCheck/custom.css" rel="stylesheet">

    <link href="css/plugins/chosen/chosen.css" rel="stylesheet">

    <link href="css/plugins/colorpicker/bootstrap-colorpicker.min.css" rel="stylesheet">

    <link href="css/plugins/cropper/cropper.min.css" rel="stylesheet">

    <link href="css/plugins/switchery/switchery.css" rel="stylesheet">

    <link href="css/plugins/jasny/jasny-bootstrap.min.css" rel="stylesheet">

    <link href="css/plugins/nouslider/jquery.nouislider.css" rel="stylesheet">

    <link href="css/plugins/datapicker/datepicker3.css" rel="stylesheet">

    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.css" rel="stylesheet">
    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.skinFlat.css" rel="stylesheet">

    <link href="css/plugins/awesome-bootstrap-checkbox/awesome-bootstrap-checkbox.css" rel="stylesheet">

    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/select2/select2.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">


    <!-- Sweet Alert -->
    <link href="css/plugins/sweetalert/sweetalert.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="active">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>
    {{$g := .Inv}}
    {{$p := .Products}}
    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <form role="form" action="/editInvoice" method="POST" onsubmit="return validateForm()" name="myForm">
                <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                    <div class="navbar-header">
                        <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i
                                class="fa fa-bars"></i>
                        </a>
                    </div>
                    <div class="minimalize-styl-2 input-group">
                        <select name="p_id" data-placeholder="Choose an Item" class="chosen-select "
                                style="width:350px;" onchange="setVal()">
                            {{range .Products}}
                            {{if .Rest}}<option value="{{.Id}},{{.Rest}},{{dec .B_p}},{{dec .S_p}}">{{.Des}} -----Buy:{{dec .B_p}}, Sell:{{dec .S_p}}, Qty.:{{.Rest}}</option>{{end}}
                            {{end}}
                        </select>
                    </div>

                    <div class="minimalize-styl-2">
                        <input hidden value="{{$g.Id}}" name="id">
                        <input hidden value="{{.NxtID}}" name="r_id">
                        <input name="b_p"  type="number" placeholder="Buy Price" value="" step="0.01">
                        <input name="s_p"  type="number" placeholder="Sell Price" value="" step="0.01">
                        <input name="qty" required type="number" placeholder="Qty" value="">
                        <input type="submit" name="submit" class="btn btn-primary " value="Add">
                    </div>
                </nav>
            </form>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-lg-12">
                    <div class="wrapper wrapper-content animated fadeInRight">
                        <div class="ibox-content p-xl">
                            <form action="/editInvoice" method="POST">
                                <div class="row">
                                    <div class="col-sm-6">
                                        <h4>Inv. No.</h4>
                                        <h4 class="text-navy">{{$g.I_no}}</h4>
                                        <span>To:</span>
                                        <address>
                                            <strong>{{($g.Cus).Name}}</strong><br>
                                            {{($g.Cus).Ad}}
                                            <br>
                                            {{($g.Cus).Phn}}
                                        </address>
                                        <p>
                                            <span><strong>Invoice Date:</strong> {{$g.Dte}}</span>
                                        </p>
                                    </div>
                                </div>

                                <div class="table-responsive m-t">
                                    <table class="table invoice-table">
                                        <thead>
                                        <tr>
                                            <th>Item List</th>
                                            <th>Buy Price</th>
                                            <th>Unit Price</th>
                                            <th>Quantity</th>
                                            <th>Total Price</th>
                                            <th>Margine</th>
                                        </tr>
                                        </thead>
                                        <tbody>

                                        {{range $g.Records}}
                                        <tr>
                                            <td>
                                                <strong>{{.P_des}}</strong>

                                                <div class="pull-right">
                                                    <a href="editInvoice?id={{$g.Id}}&r_id={{.Id}}&submit=remove"
                                                       class="btn btn-sm btn-danger">remove</a>
                                                </div>
                                            </td>
                                            <td>{{dec .B_p}}</td>
                                            <td>{{dec .S_p}}</td>
                                            <td>{{.Qty}}</td>
                                            <td>{{dec .Tot}}</td>
                                            <td>{{.Margine}}%</td>
                                        </tr>
                                        {{end}}

                                        </tbody>
                                    </table>
                                </div><!-- /table-responsive -->

                                <table class="table invoice-total">
                                    <tbody>
                                    <tr>
                                        <td><strong>Sub Total :</strong></td>
                                        <td>{{dec $g.Sub_tot}}</td>
                                    </tr>
                                    <tr>
                                        <td><strong>Vat :</strong></td>
                                        <td><input name="vat" type="number" required value="{{dec $g.Vat}}" step="0.01">
                                            <input hidden name="id" value="{{$g.Id}}">
                                        </td>
                                    </tr>
                                    <tr>
                                        <td><strong>Grand Total :</strong></td>
                                        <td>{{dec $g.Grnd_tot}}</td>
                                    </tr>
                                    <tr>
                                        <td><strong>Margine :</strong></td>
                                        <td>{{$g.Margine}}%</td>
                                    </tr>
                                    </tbody>
                                </table>
                                <div class="col-sm-12">
                                    <button type="submit" name="submit" value="Save" class="btn btn-primary pull-right">
                                        Save
                                    </button>
                                    <button type="submit" name="submit" value="Delete" class="btn btn-danger">Delete
                                        Invoice
                                    </button>
                                </div>
                                <br>
                                <input hidden value="{{$g.Id}}" name="id">

                                <div class="input-group">
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>
`+footer+`
    </div>
</div>

<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Chosen -->
<script src="js/plugins/chosen/chosen.jquery.js"></script>

<!-- JSKnob -->
<script src="js/plugins/jsKnob/jquery.knob.js"></script>

<!-- Input Mask-->
<script src="js/plugins/jasny/jasny-bootstrap.min.js"></script>

<!-- Data picker -->
<script src="js/plugins/datapicker/bootstrap-datepicker.js"></script>

<!-- NouSlider -->
<script src="js/plugins/nouslider/jquery.nouislider.min.js"></script>

<!-- Switchery -->
<script src="js/plugins/switchery/switchery.js"></script>

<!-- IonRangeSlider -->
<script src="js/plugins/ionRangeSlider/ion.rangeSlider.min.js"></script>

<!-- iCheck -->
<script src="js/plugins/iCheck/icheck.min.js"></script>

<!-- MENU -->
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>

<!-- Color picker -->
<script src="js/plugins/colorpicker/bootstrap-colorpicker.min.js"></script>

<!-- Clock picker -->
<script src="js/plugins/clockpicker/clockpicker.js"></script>

<!-- Image cropper -->
<script src="js/plugins/cropper/cropper.min.js"></script>

<!-- Date range use moment.js same as full calendar plugin -->
<script src="js/plugins/fullcalendar/moment.min.js"></script>

<!-- Date range picker -->
<script src="js/plugins/daterangepicker/daterangepicker.js"></script>

<!-- Sweet alert -->
<script src="js/plugins/sweetalert/sweetalert.min.js"></script>

<!-- Select2 -->
<script src="js/plugins/select2/select2.full.min.js"></script>
<script>
        $(document).ready(function(){

            var $image = $(".image-crop > img")
            $($image).cropper({
                aspectRatio: 1.618,
                preview: ".img-preview",
                done: function(data) {
                    // Output the result data for cropping image.
                }
            });

            var $inputImage = $("#inputImage");
            if (window.FileReader) {
                $inputImage.change(function() {
                    var fileReader = new FileReader(),
                            files = this.files,
                            file;

                    if (!files.length) {
                        return;
                    }

                    file = files[0];

                    if (/^image\/\w+$/.test(file.type)) {
                        fileReader.readAsDataURL(file);
                        fileReader.onload = function () {
                            $inputImage.val("");
                            $image.cropper("reset", true).cropper("replace", this.result);
                        };
                    } else {
                        showMessage("Please choose an image file.");
                    }
                });
            } else {
                $inputImage.addClass("hide");
            }

            $("#download").click(function() {
                window.open($image.cropper("getDataURL"));
            });

            $("#zoomIn").click(function() {
                $image.cropper("zoom", 0.1);
            });

            $("#zoomOut").click(function() {
                $image.cropper("zoom", -0.1);
            });

            $("#rotateLeft").click(function() {
                $image.cropper("rotate", 45);
            });

            $("#rotateRight").click(function() {
                $image.cropper("rotate", -45);
            });

            $("#setDrag").click(function() {
                $image.cropper("setDragMode", "crop");
            });

            $('#data_1 .input-group.date').datepicker({
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                calendarWeeks: true,
                autoclose: true
            });

            $('#data_2 .input-group.date').datepicker({
                startView: 1,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                format: "dd/mm/yyyy"
            });

            $('#data_3 .input-group.date').datepicker({
                startView: 2,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            $('#data_4 .input-group.date').datepicker({
                minViewMode: 1,
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                todayHighlight: true
            });

            $('#data_5 .input-daterange').datepicker({
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            var elem = document.querySelector('.js-switch');
            var switchery = new Switchery(elem, { color: '#1AB394' });

            var elem_2 = document.querySelector('.js-switch_2');
            var switchery_2 = new Switchery(elem_2, { color: '#ED5565' });

            var elem_3 = document.querySelector('.js-switch_3');
            var switchery_3 = new Switchery(elem_3, { color: '#1AB394' });

            $('.i-checks').iCheck({
                checkboxClass: 'icheckbox_square-green',
                radioClass: 'iradio_square-green'
            });

            $('.demo1').colorpicker();

            var divStyle = $('.back-change')[0].style;
            $('#demo_apidemo').colorpicker({
                color: divStyle.backgroundColor
            }).on('changeColor', function(ev) {
                        divStyle.backgroundColor = ev.color.toHex();
                    });

            $('.clockpicker').clockpicker();

            $('input[name="daterange"]').daterangepicker();

            $('#reportrange span').html(moment().subtract(29, 'days').format('MMMM D, YYYY') + ' - ' + moment().format('MMMM D, YYYY'));

            $('#reportrange').daterangepicker({
                format: 'MM/DD/YYYY',
                startDate: moment().subtract(29, 'days'),
                endDate: moment(),
                minDate: '01/01/2012',
                maxDate: '12/31/2015',
                dateLimit: { days: 60 },
                showDropdowns: true,
                showWeekNumbers: true,
                timePicker: false,
                timePickerIncrement: 1,
                timePicker12Hour: true,
                ranges: {
                    'Today': [moment(), moment()],
                    'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
                    'Last 7 Days': [moment().subtract(6, 'days'), moment()],
                    'Last 30 Days': [moment().subtract(29, 'days'), moment()],
                    'This Month': [moment().startOf('month'), moment().endOf('month')],
                    'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
                },
                opens: 'right',
                drops: 'down',
                buttonClasses: ['btn', 'btn-sm'],
                applyClass: 'btn-primary',
                cancelClass: 'btn-default',
                separator: ' to ',
                locale: {
                    applyLabel: 'Submit',
                    cancelLabel: 'Cancel',
                    fromLabel: 'From',
                    toLabel: 'To',
                    customRangeLabel: 'Custom',
                    daysOfWeek: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr','Sa'],
                    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
                    firstDay: 1
                }
            }, function(start, end, label) {
                console.log(start.toISOString(), end.toISOString(), label);
                $('#reportrange span').html(start.format('MMMM D, YYYY') + ' - ' + end.format('MMMM D, YYYY'));
            });

            $(".select2_demo_1").select2();
            $(".select2_demo_2").select2();
            $(".select2_demo_3").select2({
                placeholder: "Select a state",
                allowClear: true
            });


        });
        var config = {
                '.chosen-select'           : {},
                '.chosen-select-deselect'  : {allow_single_deselect:true},
                '.chosen-select-no-single' : {disable_search_threshold:10},
                '.chosen-select-no-results': {no_results_text:'Oops, nothing found!'},
                '.chosen-select-width'     : {width:"95%"}
                }
            for (var selector in config) {
                $(selector).chosen(config[selector]);
            }

        $("#ionrange_1").ionRangeSlider({
            min: 0,
            max: 5000,
            type: 'double',
            prefix: "$",
            maxPostfix: "+",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_2").ionRangeSlider({
            min: 0,
            max: 10,
            type: 'single',
            step: 0.1,
            postfix: " carats",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_3").ionRangeSlider({
            min: -50,
            max: 50,
            from: 0,
            postfix: "°",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_4").ionRangeSlider({
            values: [
                "January", "February", "March",
                "April", "May", "June",
                "July", "August", "September",
                "October", "November", "December"
            ],
            type: 'single',
            hasGrid: true
        });

        $("#ionrange_5").ionRangeSlider({
            min: 10000,
            max: 100000,
            step: 100,
            postfix: " km",
            from: 55000,
            hideMinMax: true,
            hideFromTo: false
        });

        $(".dial").knob();

        $("#basic_slider").noUiSlider({
            start: 40,
            behaviour: 'tap',
            connect: 'upper',
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#range_slider").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#drag-fixed").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag-fixed',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });



</script>

<script>
function validateForm() {
    var x = document.forms["myForm"]["p_id"].value;
    var b = document.forms["myForm"]["b_p"].value;
    var s = document.forms["myForm"]["s_p"].value;
    var qty = document.forms["myForm"]["qty"].value;
    var splits = x.split(',', 4);

    if(b==''){
    document.forms["myForm"]["b_p"].value=splits[2]
    }
    if(s==''){
    document.forms["myForm"]["s_p"].value=splits[3]
    }
    if (parseInt(splits[1])<parseInt(qty)) {
     swal("Invalied Qty.", "Quantity must be lessthan or equal to "+splits[1], "warning");
        return false;
    }
}
</script>
</body>
</html>

	`
	files["grn"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- FooTable -->
    <link href="css/plugins/footable/footable.core.css" rel="stylesheet">

    <link href="css/plugins/iCheck/custom.css" rel="stylesheet">

    <link href="css/plugins/chosen/chosen.css" rel="stylesheet">

    <link href="css/plugins/colorpicker/bootstrap-colorpicker.min.css" rel="stylesheet">

    <link href="css/plugins/cropper/cropper.min.css" rel="stylesheet">

    <link href="css/plugins/switchery/switchery.css" rel="stylesheet">

    <link href="css/plugins/jasny/jasny-bootstrap.min.css" rel="stylesheet">

    <link href="css/plugins/nouslider/jquery.nouislider.css" rel="stylesheet">

    <link href="css/plugins/datapicker/datepicker3.css" rel="stylesheet">

    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.css" rel="stylesheet">
    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.skinFlat.css" rel="stylesheet">

    <link href="css/plugins/awesome-bootstrap-checkbox/awesome-bootstrap-checkbox.css" rel="stylesheet">

    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/select2/select2.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="active">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <form role="form" action="/grn" method="POST">
                <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                    <div class="navbar-header">
                        <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i
                                class="fa fa-bars"></i>
                        </a>
                    </div>
                    <div class="minimalize-styl-2 input-group">
                        <select name="v_id" data-placeholder="Choose an Vendor" class="chosen-select "
                                style="width:350px;">
                            {{range .Vendors}}
                            <option value="{{.Id}}">{{.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="minimalize-styl-2" id="data_1">
                        <input hidden value="{{.NxtID}}" name="id">
                        <input name="g_no" required type="text" placeholder="GRN No." value="">
                        <br>
                        <div class="input-group date">
                            <span class="input-group-addon"><i class="fa fa-calendar"></i></span>
                            <input name="dte" required type="text" class="form-control" value="{{.Dte}}">
                        </div>
                    </div>
                    <div class="minimalize-styl-2">

                        <input type="submit" name="submit" class="btn btn-primary " value="Add">
                    </div>
                </nav>
            </form>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-lg-12">
                    <div class="ibox float-e-margins">

                        <div class="ibox-content">
                            <input type="text" class="form-control input-sm m-b-xs" id="filter"
                                   placeholder="Search in GRN">

                            <table class="footable table table-stripped" data-page-size="8" data-filter=#filter>
                                <thead>
                                <tr>
                                    <th>GRN No.:</th>
                                    <th>Date</th>
                                    <th>Vendor</th>
                                    <th>Grand Total</th>
                                    <th data-hide="phone,tablet">Vat</th>
                                    <th data-hide="phone,tablet">Sub Total</th>
                                </tr>
                                </thead>
                                <tbody>

                                {{range.Grns}}
                                {{$v := .Ven}}
                                <tr class="gradeX">
                                    <td>{{.G_no}}</td>
                                    <td>{{.Dte}}</td>
                                    <td>{{$v.Name}}</td>
                                    <td>{{dec .Grnd_tot}}<a class="pull-right btn btn-xs btn-info"
                                                        href="editGRN?id={{.Id}}">view/edit</a></td>
                                    <td class="center">{{dec .Vat}}</td>
                                    <td class="center">{{dec .Sub_tot}}</td>
                                </tr>
                                {{end}}

                                </tbody>
                                <tfoot>
                                <tr>
                                    <td colspan="5">
                                        <ul class="pagination pull-right"></ul>
                                    </td>
                                </tr>
                                </tfoot>
                            </table>
                        </div>
                    </div>
                </div>
            </div>
        </div>
`+footer+`
    </div>
</div>


<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>

<!-- FooTable -->
<script src="js/plugins/footable/footable.all.min.js"></script>

<!-- Page-Level Scripts -->
<script>
        $(document).ready(function() {

            $('.footable').footable();
            $('.footable2').footable();

        });

</script>


<!-- Chosen -->
<script src="js/plugins/chosen/chosen.jquery.js"></script>

<!-- JSKnob -->
<script src="js/plugins/jsKnob/jquery.knob.js"></script>

<!-- Input Mask-->
<script src="js/plugins/jasny/jasny-bootstrap.min.js"></script>

<!-- Data picker -->
<script src="js/plugins/datapicker/bootstrap-datepicker.js"></script>

<!-- NouSlider -->
<script src="js/plugins/nouslider/jquery.nouislider.min.js"></script>

<!-- Switchery -->
<script src="js/plugins/switchery/switchery.js"></script>

<!-- IonRangeSlider -->
<script src="js/plugins/ionRangeSlider/ion.rangeSlider.min.js"></script>

<!-- iCheck -->
<script src="js/plugins/iCheck/icheck.min.js"></script>

<!-- MENU -->
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>

<!-- Color picker -->
<script src="js/plugins/colorpicker/bootstrap-colorpicker.min.js"></script>

<!-- Clock picker -->
<script src="js/plugins/clockpicker/clockpicker.js"></script>

<!-- Image cropper -->
<script src="js/plugins/cropper/cropper.min.js"></script>

<!-- Date range use moment.js same as full calendar plugin -->
<script src="js/plugins/fullcalendar/moment.min.js"></script>

<!-- Date range picker -->
<script src="js/plugins/daterangepicker/daterangepicker.js"></script>

<!-- Select2 -->
<script src="js/plugins/select2/select2.full.min.js"></script>
<script>
        $(document).ready(function(){

            var $image = $(".image-crop > img")
            $($image).cropper({
                aspectRatio: 1.618,
                preview: ".img-preview",
                done: function(data) {
                    // Output the result data for cropping image.
                }
            });

            var $inputImage = $("#inputImage");
            if (window.FileReader) {
                $inputImage.change(function() {
                    var fileReader = new FileReader(),
                            files = this.files,
                            file;

                    if (!files.length) {
                        return;
                    }

                    file = files[0];

                    if (/^image\/\w+$/.test(file.type)) {
                        fileReader.readAsDataURL(file);
                        fileReader.onload = function () {
                            $inputImage.val("");
                            $image.cropper("reset", true).cropper("replace", this.result);
                        };
                    } else {
                        showMessage("Please choose an image file.");
                    }
                });
            } else {
                $inputImage.addClass("hide");
            }

            $("#download").click(function() {
                window.open($image.cropper("getDataURL"));
            });

            $("#zoomIn").click(function() {
                $image.cropper("zoom", 0.1);
            });

            $("#zoomOut").click(function() {
                $image.cropper("zoom", -0.1);
            });

            $("#rotateLeft").click(function() {
                $image.cropper("rotate", 45);
            });

            $("#rotateRight").click(function() {
                $image.cropper("rotate", -45);
            });

            $("#setDrag").click(function() {
                $image.cropper("setDragMode", "crop");
            });

            $('#data_1 .input-group.date').datepicker({
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                calendarWeeks: true,
                autoclose: true
            });

            $('#data_2 .input-group.date').datepicker({
                startView: 1,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                format: "dd/mm/yyyy"
            });

            $('#data_3 .input-group.date').datepicker({
                startView: 2,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            $('#data_4 .input-group.date').datepicker({
                minViewMode: 1,
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                todayHighlight: true
            });

            $('#data_5 .input-daterange').datepicker({
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            var elem = document.querySelector('.js-switch');
            var switchery = new Switchery(elem, { color: '#1AB394' });

            var elem_2 = document.querySelector('.js-switch_2');
            var switchery_2 = new Switchery(elem_2, { color: '#ED5565' });

            var elem_3 = document.querySelector('.js-switch_3');
            var switchery_3 = new Switchery(elem_3, { color: '#1AB394' });

            $('.i-checks').iCheck({
                checkboxClass: 'icheckbox_square-green',
                radioClass: 'iradio_square-green'
            });

            $('.demo1').colorpicker();

            var divStyle = $('.back-change')[0].style;
            $('#demo_apidemo').colorpicker({
                color: divStyle.backgroundColor
            }).on('changeColor', function(ev) {
                        divStyle.backgroundColor = ev.color.toHex();
                    });

            $('.clockpicker').clockpicker();

            $('input[name="daterange"]').daterangepicker();

            $('#reportrange span').html(moment().subtract(29, 'days').format('MMMM D, YYYY') + ' - ' + moment().format('MMMM D, YYYY'));

            $('#reportrange').daterangepicker({
                format: 'MM/DD/YYYY',
                startDate: moment().subtract(29, 'days'),
                endDate: moment(),
                minDate: '01/01/2012',
                maxDate: '12/31/2015',
                dateLimit: { days: 60 },
                showDropdowns: true,
                showWeekNumbers: true,
                timePicker: false,
                timePickerIncrement: 1,
                timePicker12Hour: true,
                ranges: {
                    'Today': [moment(), moment()],
                    'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
                    'Last 7 Days': [moment().subtract(6, 'days'), moment()],
                    'Last 30 Days': [moment().subtract(29, 'days'), moment()],
                    'This Month': [moment().startOf('month'), moment().endOf('month')],
                    'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
                },
                opens: 'right',
                drops: 'down',
                buttonClasses: ['btn', 'btn-sm'],
                applyClass: 'btn-primary',
                cancelClass: 'btn-default',
                separator: ' to ',
                locale: {
                    applyLabel: 'Submit',
                    cancelLabel: 'Cancel',
                    fromLabel: 'From',
                    toLabel: 'To',
                    customRangeLabel: 'Custom',
                    daysOfWeek: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr','Sa'],
                    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
                    firstDay: 1
                }
            }, function(start, end, label) {
                console.log(start.toISOString(), end.toISOString(), label);
                $('#reportrange span').html(start.format('MMMM D, YYYY') + ' - ' + end.format('MMMM D, YYYY'));
            });

            $(".select2_demo_1").select2();
            $(".select2_demo_2").select2();
            $(".select2_demo_3").select2({
                placeholder: "Select a state",
                allowClear: true
            });


        });
        var config = {
                '.chosen-select'           : {},
                '.chosen-select-deselect'  : {allow_single_deselect:true},
                '.chosen-select-no-single' : {disable_search_threshold:10},
                '.chosen-select-no-results': {no_results_text:'Oops, nothing found!'},
                '.chosen-select-width'     : {width:"95%"}
                }
            for (var selector in config) {
                $(selector).chosen(config[selector]);
            }

        $("#ionrange_1").ionRangeSlider({
            min: 0,
            max: 5000,
            type: 'double',
            prefix: "$",
            maxPostfix: "+",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_2").ionRangeSlider({
            min: 0,
            max: 10,
            type: 'single',
            step: 0.1,
            postfix: " carats",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_3").ionRangeSlider({
            min: -50,
            max: 50,
            from: 0,
            postfix: "°",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_4").ionRangeSlider({
            values: [
                "January", "February", "March",
                "April", "May", "June",
                "July", "August", "September",
                "October", "November", "December"
            ],
            type: 'single',
            hasGrid: true
        });

        $("#ionrange_5").ionRangeSlider({
            min: 10000,
            max: 100000,
            step: 100,
            postfix: " km",
            from: 55000,
            hideMinMax: true,
            hideFromTo: false
        });

        $(".dial").knob();

        $("#basic_slider").noUiSlider({
            start: 40,
            behaviour: 'tap',
            connect: 'upper',
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#range_slider").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#drag-fixed").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag-fixed',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });
</script>


</body>
</html>

	`
	files["home"] = `
	<!DOCTYPE html>
<html>

<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>mandy</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

</head>

<body class="gray-bg">
    <div class="middle-box text-center loginscreen animated fadeInDown">
        <div>



            <h2>Select Business</h2>
            <form class="m-t" role="form" action="home">
                <input type="submit" class="btn btn-primary block full-width m-b btn-xl" value="Consumer" name="submit">
                <input type="submit" class="btn btn-primary block full-width m-b btn-xl" value="Bakery" name="submit">
            </form>
        </div>
    </div>
`+footer+`
    <!-- Mainly scripts -->
    <script src="js/jquery-2.1.1.js"></script>
    <script src="js/bootstrap.min.js"></script>

</body>

</html>

	`

	files["invoice"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- FooTable -->
    <link href="css/plugins/footable/footable.core.css" rel="stylesheet">

    <link href="css/plugins/iCheck/custom.css" rel="stylesheet">

    <link href="css/plugins/chosen/chosen.css" rel="stylesheet">

    <link href="css/plugins/colorpicker/bootstrap-colorpicker.min.css" rel="stylesheet">

    <link href="css/plugins/cropper/cropper.min.css" rel="stylesheet">

    <link href="css/plugins/switchery/switchery.css" rel="stylesheet">

    <link href="css/plugins/jasny/jasny-bootstrap.min.css" rel="stylesheet">

    <link href="css/plugins/nouslider/jquery.nouislider.css" rel="stylesheet">

    <link href="css/plugins/datapicker/datepicker3.css" rel="stylesheet">

    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.css" rel="stylesheet">
    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.skinFlat.css" rel="stylesheet">

    <link href="css/plugins/awesome-bootstrap-checkbox/awesome-bootstrap-checkbox.css" rel="stylesheet">

    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/select2/select2.min.css" rel="stylesheet">


</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="active">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <form role="form" action="/invoice" method="POST">
                <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                    <div class="navbar-header">
                        <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i
                                class="fa fa-bars"></i>
                        </a>
                    </div>
                    <div class="minimalize-styl-2 input-group">
                        <select name="c_id" data-placeholder="Choose a Customer" class="chosen-select "
                                style="width:350px;">
                            {{range .Cus}}
                            <option value="{{.Id}}">{{.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="minimalize-styl-2 input-group">
                        <select name="v_id" data-placeholder="Choose a Van" class="chosen-select "
                                style="width:350px;">
                            {{range .Vans}}
                            <option value="{{.Id}}">{{.Des}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="minimalize-styl-2" id="data_1">
                        <input hidden value="{{.NxtID}}" name="id">
                        <input name="i_no" required type="text" placeholder="Inv. No." value="">
                        <input name="po_no" required type="text" placeholder="Po. No." value="">
                        <br>
                        <div class="input-group date">
                            <span class="input-group-addon"><i class="fa fa-calendar"></i></span>
                            <input name="dte" required type="text" class="form-control" value="{{.Dte}}">
                        </div>
                    </div>
                    <div class="minimalize-styl-2">

                        <input type="submit" name="submit" class="btn btn-primary " value="Add">
                    </div>
                </nav>
            </form>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-12">
                    <div class="ibox">
                        <div class="ibox-content">
                            <h2>Vehicles</h2>
                            <div class="input-group">
                                <input type="text" class="input form-control"
                                       id="filter"
                                       placeholder="Search in Vehicles">
                                </span>
                            </div>
                            <div class="clients-list">
                                <ul class="nav nav-tabs">
                                    {{range .Vans}}
                                    <li {{if .Active}}class="active" {{end}}><a data-toggle="tab" href="#van{{.Id}}"><i
                                            class="fa fa-bus"></i>
                                        {{.Des}}</a>
                                    </li>
                                    {{end}}
                                </ul>
                                <div class="tab-content">
                                    {{$newid := .NxtID}}
                                    {{range .Vans}}
                                    <div id="van{{.Id}}" class="tab-pane {{if .Active}}active{{end}}">
                                        <div class="full-height-scroll">
                                            <div class="col-lg-12">
                                                <div class="ibox float-e-margins">
                                                    <div class="ibox-content">
                                                        <table class="footable table table-stripped" data-page-size="8"
                                                               data-filter=#filter>
                                                            <thead>
                                                            <tr>
                                                                <th>Date</th>
                                                                <th>Inv. No.</th>
                                                                <th>Cus. Name</th>
                                                                <th>Po. No.</th>
                                                                <th>G. Tot</th>
                                                                <th>Payments</th>
                                                                <th>Margine</th>
                                                            </tr>
                                                            </thead>
                                                            <tbody>
                                                            {{range .Invoices}}
                                                            {{$c:=.Cus}}
                                                            <tr>
                                                                <td>{{.Dte}}</td>
                                                                <td>{{.I_no}}</td>
                                                                <td>{{$c.Name}}</td>
                                                                <td>{{.Po_no}}</td>
                                                                <td>{{dec .Grnd_tot}}</td>
                                                                <td>{{dec .PaymentsDone}}</td>
                                                                <td>{{.Margine}}
                                                                    <a class="pull-right btn btn-xs btn-info"
                                                                       href="editInvoice?id={{.Id}}">view/edit</a>
                                                                </td>
                                                            </tr>
                                                            {{end}}
                                                            </tbody>
                                                            <tfoot>
                                                            <tr>
                                                                <td colspan="5">
                                                                    <ul class="pagination pull-right"></ul>
                                                                </td>
                                                            </tr>
                                                            </tfoot>
                                                        </table>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>


                                    </div>
                                    {{end}}
                                </div>

                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
`+footer+`
    </div>
</div>


<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>

<!-- FooTable -->
<script src="js/plugins/footable/footable.all.min.js"></script>

<!-- Page-Level Scripts -->
<script>
        $(document).ready(function() {

            $('.footable').footable();
            $('.footable2').footable();

        });

</script>


<!-- Chosen -->
<script src="js/plugins/chosen/chosen.jquery.js"></script>

<!-- JSKnob -->
<script src="js/plugins/jsKnob/jquery.knob.js"></script>

<!-- Input Mask-->
<script src="js/plugins/jasny/jasny-bootstrap.min.js"></script>

<!-- Data picker -->
<script src="js/plugins/datapicker/bootstrap-datepicker.js"></script>

<!-- NouSlider -->
<script src="js/plugins/nouslider/jquery.nouislider.min.js"></script>

<!-- Switchery -->
<script src="js/plugins/switchery/switchery.js"></script>

<!-- IonRangeSlider -->
<script src="js/plugins/ionRangeSlider/ion.rangeSlider.min.js"></script>

<!-- iCheck -->
<script src="js/plugins/iCheck/icheck.min.js"></script>

<!-- MENU -->
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>

<!-- Color picker -->
<script src="js/plugins/colorpicker/bootstrap-colorpicker.min.js"></script>

<!-- Clock picker -->
<script src="js/plugins/clockpicker/clockpicker.js"></script>

<!-- Image cropper -->
<script src="js/plugins/cropper/cropper.min.js"></script>

<!-- Date range use moment.js same as full calendar plugin -->
<script src="js/plugins/fullcalendar/moment.min.js"></script>

<!-- Date range picker -->
<script src="js/plugins/daterangepicker/daterangepicker.js"></script>

<!-- Select2 -->
<script src="js/plugins/select2/select2.full.min.js"></script>
<script>
        $(document).ready(function(){

            var $image = $(".image-crop > img")
            $($image).cropper({
                aspectRatio: 1.618,
                preview: ".img-preview",
                done: function(data) {
                    // Output the result data for cropping image.
                }
            });

            var $inputImage = $("#inputImage");
            if (window.FileReader) {
                $inputImage.change(function() {
                    var fileReader = new FileReader(),
                            files = this.files,
                            file;

                    if (!files.length) {
                        return;
                    }

                    file = files[0];

                    if (/^image\/\w+$/.test(file.type)) {
                        fileReader.readAsDataURL(file);
                        fileReader.onload = function () {
                            $inputImage.val("");
                            $image.cropper("reset", true).cropper("replace", this.result);
                        };
                    } else {
                        showMessage("Please choose an image file.");
                    }
                });
            } else {
                $inputImage.addClass("hide");
            }

            $("#download").click(function() {
                window.open($image.cropper("getDataURL"));
            });

            $("#zoomIn").click(function() {
                $image.cropper("zoom", 0.1);
            });

            $("#zoomOut").click(function() {
                $image.cropper("zoom", -0.1);
            });

            $("#rotateLeft").click(function() {
                $image.cropper("rotate", 45);
            });

            $("#rotateRight").click(function() {
                $image.cropper("rotate", -45);
            });

            $("#setDrag").click(function() {
                $image.cropper("setDragMode", "crop");
            });

            $('#data_1 .input-group.date').datepicker({
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                calendarWeeks: true,
                autoclose: true
            });

            $('#data_2 .input-group.date').datepicker({
                startView: 1,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                format: "dd/mm/yyyy"
            });

            $('#data_3 .input-group.date').datepicker({
                startView: 2,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            $('#data_4 .input-group.date').datepicker({
                minViewMode: 1,
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                todayHighlight: true
            });

            $('#data_5 .input-daterange').datepicker({
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            var elem = document.querySelector('.js-switch');
            var switchery = new Switchery(elem, { color: '#1AB394' });

            var elem_2 = document.querySelector('.js-switch_2');
            var switchery_2 = new Switchery(elem_2, { color: '#ED5565' });

            var elem_3 = document.querySelector('.js-switch_3');
            var switchery_3 = new Switchery(elem_3, { color: '#1AB394' });

            $('.i-checks').iCheck({
                checkboxClass: 'icheckbox_square-green',
                radioClass: 'iradio_square-green'
            });

            $('.demo1').colorpicker();

            var divStyle = $('.back-change')[0].style;
            $('#demo_apidemo').colorpicker({
                color: divStyle.backgroundColor
            }).on('changeColor', function(ev) {
                        divStyle.backgroundColor = ev.color.toHex();
                    });

            $('.clockpicker').clockpicker();

            $('input[name="daterange"]').daterangepicker();

            $('#reportrange span').html(moment().subtract(29, 'days').format('MMMM D, YYYY') + ' - ' + moment().format('MMMM D, YYYY'));

            $('#reportrange').daterangepicker({
                format: 'MM/DD/YYYY',
                startDate: moment().subtract(29, 'days'),
                endDate: moment(),
                minDate: '01/01/2012',
                maxDate: '12/31/2015',
                dateLimit: { days: 60 },
                showDropdowns: true,
                showWeekNumbers: true,
                timePicker: false,
                timePickerIncrement: 1,
                timePicker12Hour: true,
                ranges: {
                    'Today': [moment(), moment()],
                    'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
                    'Last 7 Days': [moment().subtract(6, 'days'), moment()],
                    'Last 30 Days': [moment().subtract(29, 'days'), moment()],
                    'This Month': [moment().startOf('month'), moment().endOf('month')],
                    'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
                },
                opens: 'right',
                drops: 'down',
                buttonClasses: ['btn', 'btn-sm'],
                applyClass: 'btn-primary',
                cancelClass: 'btn-default',
                separator: ' to ',
                locale: {
                    applyLabel: 'Submit',
                    cancelLabel: 'Cancel',
                    fromLabel: 'From',
                    toLabel: 'To',
                    customRangeLabel: 'Custom',
                    daysOfWeek: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr','Sa'],
                    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
                    firstDay: 1
                }
            }, function(start, end, label) {
                console.log(start.toISOString(), end.toISOString(), label);
                $('#reportrange span').html(start.format('MMMM D, YYYY') + ' - ' + end.format('MMMM D, YYYY'));
            });

            $(".select2_demo_1").select2();
            $(".select2_demo_2").select2();
            $(".select2_demo_3").select2({
                placeholder: "Select a state",
                allowClear: true
            });


        });
        var config = {
                '.chosen-select'           : {},
                '.chosen-select-deselect'  : {allow_single_deselect:true},
                '.chosen-select-no-single' : {disable_search_threshold:10},
                '.chosen-select-no-results': {no_results_text:'Oops, nothing found!'},
                '.chosen-select-width'     : {width:"95%"}
                }
            for (var selector in config) {
                $(selector).chosen(config[selector]);
            }

        $("#ionrange_1").ionRangeSlider({
            min: 0,
            max: 5000,
            type: 'double',
            prefix: "$",
            maxPostfix: "+",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_2").ionRangeSlider({
            min: 0,
            max: 10,
            type: 'single',
            step: 0.1,
            postfix: " carats",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_3").ionRangeSlider({
            min: -50,
            max: 50,
            from: 0,
            postfix: "°",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_4").ionRangeSlider({
            values: [
                "January", "February", "March",
                "April", "May", "June",
                "July", "August", "September",
                "October", "November", "December"
            ],
            type: 'single',
            hasGrid: true
        });

        $("#ionrange_5").ionRangeSlider({
            min: 10000,
            max: 100000,
            step: 100,
            postfix: " km",
            from: 55000,
            hideMinMax: true,
            hideFromTo: false
        });

        $(".dial").knob();

        $("#basic_slider").noUiSlider({
            start: 40,
            behaviour: 'tap',
            connect: 'upper',
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#range_slider").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#drag-fixed").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag-fixed',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

</script>

</body>
</html>

	`
	files["load"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- FooTable -->
    <link href="css/plugins/footable/footable.core.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="active">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                <div class="navbar-header">
                    <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i class="fa fa-bars"></i>
                    </a>
                </div>

            </nav>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-12">
                    <div class="ibox">
                        <div class="ibox-content">
                            <h2>Vehicles</h2>
                            <div class="input-group">
                                <input type="text" class="input form-control"
                                       id="filter"
                                       placeholder="Search in Loading">
                                </span>
                            </div>
                            <div class="clients-list">
                                <ul class="nav nav-tabs">
                                    {{range .Vans}}
                                    <li {{if .Active}}class="active" {{end}}><a data-toggle="tab" href="#van{{.Id}}"><i
                                            class="fa fa-bus"></i>
                                        {{.Des}}</a></li>
                                    {{end}}
                                </ul>
                                <div class="tab-content">
                                    {{range .Vans}}
                                    <div id="van{{.Id}}" class="tab-pane {{if .Active}}active{{end}}">
                                        <div class="full-height-scroll">
                                            <div class="col-lg-12">
                                                <div class="ibox float-e-margins">
                                                    <div class="ibox-content">
                                                        <table class="footable table table-stripped" data-page-size="8"
                                                               data-filter=#filter>
                                                            <thead>
                                                            <tr>
                                                                <th>Date</th>
                                                                <th>Product</th>
                                                                <th>Qty.</th>
                                                            </tr>
                                                            </thead>
                                                            <tbody>
                                                            {{range .Loads}}
                                                            <tr class="gradeU">
                                                                <td>{{.Dte}}</td>
                                                                <td>{{.P_des}}</td>
                                                                <td>{{.Qty}}</td>
                                                            </tr>
                                                            {{end}}
                                                            </tbody>
                                                            <tfoot>
                                                            <tr>
                                                                <td colspan="5">
                                                                    <ul class="pagination pull-right"></ul>
                                                                </td>
                                                            </tr>
                                                            </tfoot>
                                                        </table>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>


                                    </div>
                                    {{end}}
                                </div>

                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
`+footer+`
    </div>
</div>


<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>


<!-- FooTable -->
<script src="js/plugins/footable/footable.all.min.js"></script>


<!-- Page-Level Scripts -->
<script>
        $(document).ready(function() {

            $('.footable').footable();
            $('.footable2').footable();

        });




</script>
</body>
</html>

	`
	files["payment"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- FooTable -->
    <link href="css/plugins/footable/footable.core.css" rel="stylesheet">


    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/iCheck/custom.css" rel="stylesheet">

    <link href="css/plugins/chosen/chosen.css" rel="stylesheet">

    <link href="css/plugins/colorpicker/bootstrap-colorpicker.min.css" rel="stylesheet">

    <link href="css/plugins/cropper/cropper.min.css" rel="stylesheet">

    <link href="css/plugins/switchery/switchery.css" rel="stylesheet">

    <link href="css/plugins/jasny/jasny-bootstrap.min.css" rel="stylesheet">

    <link href="css/plugins/nouslider/jquery.nouislider.css" rel="stylesheet">

    <link href="css/plugins/datapicker/datepicker3.css" rel="stylesheet">

    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.css" rel="stylesheet">
    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.skinFlat.css" rel="stylesheet">

    <link href="css/plugins/awesome-bootstrap-checkbox/awesome-bootstrap-checkbox.css" rel="stylesheet">

    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/select2/select2.min.css" rel="stylesheet">


    <!-- Sweet Alert -->
    <link href="css/plugins/sweetalert/sweetalert.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="active">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <form role="form" action="/payment" method="POST" onsubmit="return validateForm()" name="myForm">
                <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                    <div class="navbar-header">
                        <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i
                                class="fa fa-bars"></i>
                        </a>
                    </div>
                    <div class="minimalize-styl-2 input-group">
                        <select name="i_id" data-placeholder="Choose an Invoice" class="chosen-select "
                                style="width:350px;">
                            {{range .Invoices}}
                            {{if .RemainingPayment}}
                            <option value="{{.Id}},{{.RemainingPayment}}">{{(.Cus).Name}} :-
                                Inv.No.:{{.I_no}}_Po.No.:{{.Po_no}}_Due:{{dec .RemainingPayment}}
                            </option>
                            {{end}}
                            {{end}}
                        </select>
                    </div>
                    <div class="minimalize-styl-2" id="data_1">
                        <input name="id" hidden value="{{.Nid}}">
                        <input name="des" required type="text" placeholder="Description" value="CASH">
                        <input name="tot" required type="number" placeholder="Payment" value="" step="0.01">
                        <div class="input-group date">
                            <span class="input-group-addon"><i class="fa fa-calendar"></i></span>
                            <input name="dte" required type="text" class="form-control" value="{{.Dte}}">
                        </div>
                    </div>
                    <div class="minimalize-styl-2">

                        <input type="submit" name="submit" class="btn btn-primary " value="Add">
                    </div>
                </nav>
            </form>


        </div>


        {{range .Payments}}
        <div id="p{{.Id}}" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form role="form" action="/payment" method="POST">
                                    <input hidden name="id" value="{{.Id}}">
                                    <div class="form-group"><label>Description</label>
                                        <input name="des" required type="text" placeholder="Description"
                                               class="form-control" value="{{.Des}}">
                                    </div>
                                    <div class="form-group"><label>Payment</label>
                                        <input name="tot" required type="number" placeholder="Payment"
                                               class="form-control" value="{{dec .Tot}}" step="0.01">
                                    </div>
                                    <div class="form-group" id="data_1"><label>Date</label>
                                        <div class="input-group date">
                                            <span class="input-group-addon"><i class="fa fa-calendar"></i></span>
                                            <input name="dte" required type="text" class="form-control"
                                                   value="{{.Dte}}">
                                        </div>
                                    </div>
                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs" value="Save"></strong>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-danger pull-right m-t-n-xs" value="Delete"></strong>
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        {{end}}

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-12">
                    <div class="ibox float-e-margins">
                        <div class="ibox-content">
                            <input type="text" class="form-control input-sm m-b-xs" id="filter"
                                   placeholder="Search in table">

                            <table class="footable table table-stripped" data-page-size="8" data-filter=#filter>
                                <thead>
                                <tr>
                                    <th>Date</th>
                                    <th>Inv No.</th>
                                    <th>Customer</th>
                                    <th>Total</th>
                                    <th>Description</th>
                                    <th></th>
                                </tr>
                                </thead>
                                <tbody>
                                {{range.Payments}}
                                <tr>
                                    <td>{{.Dte}}</td>
                                    <td>{{.I_no}}</td>
                                    <td>{{.C_name}}</td>
                                    <td>{{dec .Tot}}</td>
                                    <td>{{.Des}}</td>
                                    <th>
                                        <a data-toggle="modal" class="pull-right btn btn-xs btn-warning"
                                           href="#p{{.Id}}">edit</a>
                                    </th>
                                </tr>
                                {{end}}
                                </tbody>
                                <tfoot>
                                <tr>
                                    <td colspan="5">
                                        <ul class="pagination pull-right"></ul>
                                    </td>
                                </tr>
                                </tfoot>
                            </table>
                        </div>
                    </div>
                </div>
            </div>
        </div>
`+footer+`
    </div>
</div>


<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>

<!-- FooTable -->
<script src="js/plugins/footable/footable.all.min.js"></script>

<!-- Page-Level Scripts -->
<script>
        $(document).ready(function() {

            $('.footable').footable();
            $('.footable2').footable();

        });

</script>


<!-- Chosen -->
<script src="js/plugins/chosen/chosen.jquery.js"></script>

<!-- JSKnob -->
<script src="js/plugins/jsKnob/jquery.knob.js"></script>

<!-- Input Mask-->
<script src="js/plugins/jasny/jasny-bootstrap.min.js"></script>

<!-- Data picker -->
<script src="js/plugins/datapicker/bootstrap-datepicker.js"></script>

<!-- NouSlider -->
<script src="js/plugins/nouslider/jquery.nouislider.min.js"></script>

<!-- Switchery -->
<script src="js/plugins/switchery/switchery.js"></script>

<!-- IonRangeSlider -->
<script src="js/plugins/ionRangeSlider/ion.rangeSlider.min.js"></script>

<!-- iCheck -->
<script src="js/plugins/iCheck/icheck.min.js"></script>

<!-- MENU -->
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>

<!-- Color picker -->
<script src="js/plugins/colorpicker/bootstrap-colorpicker.min.js"></script>

<!-- Clock picker -->
<script src="js/plugins/clockpicker/clockpicker.js"></script>

<!-- Image cropper -->
<script src="js/plugins/cropper/cropper.min.js"></script>

<!-- Date range use moment.js same as full calendar plugin -->
<script src="js/plugins/fullcalendar/moment.min.js"></script>

<!-- Date range picker -->
<script src="js/plugins/daterangepicker/daterangepicker.js"></script>

<!-- Select2 -->
<script src="js/plugins/select2/select2.full.min.js"></script>

<!-- Sweet alert -->
<script src="js/plugins/sweetalert/sweetalert.min.js"></script>

<script>
        $(document).ready(function(){

            var $image = $(".image-crop > img")
            $($image).cropper({
                aspectRatio: 1.618,
                preview: ".img-preview",
                done: function(data) {
                    // Output the result data for cropping image.
                }
            });

            var $inputImage = $("#inputImage");
            if (window.FileReader) {
                $inputImage.change(function() {
                    var fileReader = new FileReader(),
                            files = this.files,
                            file;

                    if (!files.length) {
                        return;
                    }

                    file = files[0];

                    if (/^image\/\w+$/.test(file.type)) {
                        fileReader.readAsDataURL(file);
                        fileReader.onload = function () {
                            $inputImage.val("");
                            $image.cropper("reset", true).cropper("replace", this.result);
                        };
                    } else {
                        showMessage("Please choose an image file.");
                    }
                });
            } else {
                $inputImage.addClass("hide");
            }

            $("#download").click(function() {
                window.open($image.cropper("getDataURL"));
            });

            $("#zoomIn").click(function() {
                $image.cropper("zoom", 0.1);
            });

            $("#zoomOut").click(function() {
                $image.cropper("zoom", -0.1);
            });

            $("#rotateLeft").click(function() {
                $image.cropper("rotate", 45);
            });

            $("#rotateRight").click(function() {
                $image.cropper("rotate", -45);
            });

            $("#setDrag").click(function() {
                $image.cropper("setDragMode", "crop");
            });

            $('#data_1 .input-group.date').datepicker({
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                calendarWeeks: true,
                autoclose: true
            });

            $('#data_2 .input-group.date').datepicker({
                startView: 1,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                format: "dd/mm/yyyy"
            });

            $('#data_3 .input-group.date').datepicker({
                startView: 2,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            $('#data_4 .input-group.date').datepicker({
                minViewMode: 1,
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                todayHighlight: true
            });

            $('#data_5 .input-daterange').datepicker({
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            var elem = document.querySelector('.js-switch');
            var switchery = new Switchery(elem, { color: '#1AB394' });

            var elem_2 = document.querySelector('.js-switch_2');
            var switchery_2 = new Switchery(elem_2, { color: '#ED5565' });

            var elem_3 = document.querySelector('.js-switch_3');
            var switchery_3 = new Switchery(elem_3, { color: '#1AB394' });

            $('.i-checks').iCheck({
                checkboxClass: 'icheckbox_square-green',
                radioClass: 'iradio_square-green'
            });

            $('.demo1').colorpicker();

            var divStyle = $('.back-change')[0].style;
            $('#demo_apidemo').colorpicker({
                color: divStyle.backgroundColor
            }).on('changeColor', function(ev) {
                        divStyle.backgroundColor = ev.color.toHex();
                    });

            $('.clockpicker').clockpicker();

            $('input[name="daterange"]').daterangepicker();

            $('#reportrange span').html(moment().subtract(29, 'days').format('MMMM D, YYYY') + ' - ' + moment().format('MMMM D, YYYY'));

            $('#reportrange').daterangepicker({
                format: 'MM/DD/YYYY',
                startDate: moment().subtract(29, 'days'),
                endDate: moment(),
                minDate: '01/01/2012',
                maxDate: '12/31/2015',
                dateLimit: { days: 60 },
                showDropdowns: true,
                showWeekNumbers: true,
                timePicker: false,
                timePickerIncrement: 1,
                timePicker12Hour: true,
                ranges: {
                    'Today': [moment(), moment()],
                    'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
                    'Last 7 Days': [moment().subtract(6, 'days'), moment()],
                    'Last 30 Days': [moment().subtract(29, 'days'), moment()],
                    'This Month': [moment().startOf('month'), moment().endOf('month')],
                    'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
                },
                opens: 'right',
                drops: 'down',
                buttonClasses: ['btn', 'btn-sm'],
                applyClass: 'btn-primary',
                cancelClass: 'btn-default',
                separator: ' to ',
                locale: {
                    applyLabel: 'Submit',
                    cancelLabel: 'Cancel',
                    fromLabel: 'From',
                    toLabel: 'To',
                    customRangeLabel: 'Custom',
                    daysOfWeek: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr','Sa'],
                    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
                    firstDay: 1
                }
            }, function(start, end, label) {
                console.log(start.toISOString(), end.toISOString(), label);
                $('#reportrange span').html(start.format('MMMM D, YYYY') + ' - ' + end.format('MMMM D, YYYY'));
            });

            $(".select2_demo_1").select2();
            $(".select2_demo_2").select2();
            $(".select2_demo_3").select2({
                placeholder: "Select a state",
                allowClear: true
            });


        });
        var config = {
                '.chosen-select'           : {},
                '.chosen-select-deselect'  : {allow_single_deselect:true},
                '.chosen-select-no-single' : {disable_search_threshold:10},
                '.chosen-select-no-results': {no_results_text:'Oops, nothing found!'},
                '.chosen-select-width'     : {width:"95%"}
                }
            for (var selector in config) {
                $(selector).chosen(config[selector]);
            }

        $("#ionrange_1").ionRangeSlider({
            min: 0,
            max: 5000,
            type: 'double',
            prefix: "$",
            maxPostfix: "+",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_2").ionRangeSlider({
            min: 0,
            max: 10,
            type: 'single',
            step: 0.1,
            postfix: " carats",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_3").ionRangeSlider({
            min: -50,
            max: 50,
            from: 0,
            postfix: "°",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_4").ionRangeSlider({
            values: [
                "January", "February", "March",
                "April", "May", "June",
                "July", "August", "September",
                "October", "November", "December"
            ],
            type: 'single',
            hasGrid: true
        });

        $("#ionrange_5").ionRangeSlider({
            min: 10000,
            max: 100000,
            step: 100,
            postfix: " km",
            from: 55000,
            hideMinMax: true,
            hideFromTo: false
        });

        $(".dial").knob();

        $("#basic_slider").noUiSlider({
            start: 40,
            behaviour: 'tap',
            connect: 'upper',
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#range_slider").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#drag-fixed").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag-fixed',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });
</script>
<script>
function validateForm() {
    var x = document.forms["myForm"]["i_id"].value;
    var qty = document.forms["myForm"]["tot"].value;
    var splits = x.split(',', 2);

    if (parseInt(splits[1])<parseInt(qty)) {
     swal("Invalied Payment", "Payment must be lessthan or equal to "+splits[1], "warning");
        return false;
    }
}

</script>
</body>
</html>

	`
	files["products"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Data Tables -->
    <link href="css/plugins/dataTables/dataTables.bootstrap.css" rel="stylesheet">
    <link href="css/plugins/dataTables/dataTables.responsive.css" rel="stylesheet">
    <link href="css/plugins/dataTables/dataTables.tableTools.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="active">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                <div class="navbar-header">
                    <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i class="fa fa-bars"></i>
                    </a>
                </div>
                <div class="">
                    <a data-toggle="modal" href="#Addnew" class="btn btn-primary minimalize-styl-2">Add new</a>
                </div>
            </nav>
        </div>

        {{range .Products}}
        <div id="p{{.Id}}" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form role="form" action="/products" method="POST" class="cform">
                                    <div class="form-group"><label>ID</label>
                                        <input name="id" readonly required type="text" placeholder="ID"
                                               class="form-control" value="{{.Id}}">
                                    </div>
                                    <div class="form-group"><label>Description</label>
                                        <input name="des" required type="text" placeholder="Description"
                                               class="form-control" value="{{.Des}}">
                                    </div>
                                    <div class="form-group"><label>Buy Price</label>
                                        <input name="b_p" required type="number" placeholder="Buy Price"
                                               class="form-control" value="{{dec .B_p}}">
                                    </div>
                                    <div class="form-group"><label>Sell Price</label>
                                        <input name="s_p" required type="number" placeholder="Sell Price"
                                               class="form-control" value="{{dec .S_p}}">
                                    </div>
                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs" value="Save"></strong>
                                        {{if .DeleteBTN}}
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-danger pull-right m-t-n-xs" value="Delete"></strong>
                                        {{end}}
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        {{end}}


        <div id="Addnew" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form class="cform" role="form" action="/products" method="POST">
                                    <div class="form-group"><label>ID</label>
                                        <input name="id" readonly required type="text" placeholder="ID"
                                               class="form-control" value="{{.Nid}}">
                                    </div>
                                    <div class="form-group"><label>Description</label>
                                        <input name="des" required type="text" placeholder="Description"
                                               class="form-control" value="">
                                    </div>
                                    <div class="form-group"><label>Buy Price</label>
                                        <input name="b_p" required type="number" placeholder="Buy Price"
                                               class="form-control" value="0">
                                    </div>
                                    <div class="form-group"><label>Sell Price</label>
                                        <input name="s_p" required type="number" placeholder="Sell Price"
                                               class="form-control" value="0">
                                    </div>
                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs"
                                                       value="Add"></strong>
                                        <strong><input type="reset" class="btn btn-sm btn-warning pull-right m-t-n-xs"
                                                       value="Reset"></strong>
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>


        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="ibox-content m-b-sm border-bottom">
                <div class="row">
                    <div class="col-lg-12">
                        <div class="ibox float-e-margins">
                            <div class="ibox-content">
                                <div class="table-responsive">
                                    <table class="table table-striped table-bordered table-hover dataTables-example">
                                        <thead>
                                        <tr>
                                            <th>Description</th>
                                            <th>Buy Price</th>
                                            <th>Sell Price</th>
                                            <th>Main Stock</th>
                                            <th>Vehi. Stock</th>
                                            <th>Total Qty</th>
                                            <th>Option</th>
                                        </tr>
                                        </thead>
                                        <tbody>
                                        {{range .Products}}
                                        <tr class="gradeA">
                                            <td>{{.Des}}</td>
                                            <td>{{dec .B_p }}</td>
                                            <td>{{dec .S_p}}</td>
                                            <td>{{.QtyStk}}</td>
                                            <td>{{.QtyVan}}</td>
                                            <td>{{.Qty}}</td>
                                            <td><a data-toggle="modal" class="btn btn-info btn-sm btn-outline"
                                                   href="#p{{.Id}}">View
                                                / Edit</a></td>
                                        </tr>
                                        {{end}}
                                        </tbody>
                                        <tfoot>
                                        <tr>
                                            <th>Description</th>
                                            <th>Buy Price</th>
                                            <th>Sell Price</th>
                                            <th>Main Stock</th>
                                            <th>Vehi. Stock</th>
                                            <th>Total Qty</th>
                                            <th>Option</th>
                                        </tr>
                                        </tfoot>
                                    </table>
                                </div>

                            </div>
                        </div>
                    </div>

                </div>

            </div>

        </div>
        `+footer+`
    </div>

    <!-- Mainly scripts -->
    <script src="js/jquery-2.1.1.js"></script>
    <script src="js/bootstrap.min.js"></script>
    <script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
    <script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>
    <script src="js/plugins/jeditable/jquery.jeditable.js"></script>

    <!-- Data Tables -->
    <script src="js/plugins/dataTables/jquery.dataTables.js"></script>
    <script src="js/plugins/dataTables/dataTables.bootstrap.js"></script>
    <script src="js/plugins/dataTables/dataTables.responsive.js"></script>
    <script src="js/plugins/dataTables/dataTables.tableTools.min.js"></script>

    <!-- Custom and plugin javascript -->
    <script src="js/inspinia.js"></script>
    <script src="js/plugins/pace/pace.min.js"></script>

    <style>

    </style>

    <!-- Page-Level Scripts -->
    <script>
        $(document).ready(function() {
            $('.dataTables-example').DataTable({

            });


        });

    </script>
 <!-- Jquery Validate -->
    <script src="js/plugins/validate/jquery.validate.min.js"></script>

    <script>
         $(document).ready(function(){

             $(".cform").validate({
                 rules: {
                     password: {
                         required: true,
                         minlength: 3
                     },
                     url: {
                         required: true,
                         url: true
                     },
                     number: {
                         required: true,
                         number: true
                     },
                     min: {
                         required: true,
                         minlength: 6
                     },
                     max: {
                         required: true,
                         maxlength: 4
                     }
                 }
             });
        });
    </script>
</body>
</html>

	`

	files["stat"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- orris -->
    <link href="css/plugins/morris/morris-0.4.3.min.css" rel="stylesheet">

    <link href="css/plugins/iCheck/custom.css" rel="stylesheet">

    <link href="css/plugins/chosen/chosen.css" rel="stylesheet">

    <link href="css/plugins/colorpicker/bootstrap-colorpicker.min.css" rel="stylesheet">

    <link href="css/plugins/cropper/cropper.min.css" rel="stylesheet">

    <link href="css/plugins/switchery/switchery.css" rel="stylesheet">

    <link href="css/plugins/jasny/jasny-bootstrap.min.css" rel="stylesheet">

    <link href="css/plugins/nouslider/jquery.nouislider.css" rel="stylesheet">

    <link href="css/plugins/datapicker/datepicker3.css" rel="stylesheet">

    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.css" rel="stylesheet">
    <link href="css/plugins/ionRangeSlider/ion.rangeSlider.skinFlat.css" rel="stylesheet">

    <link href="css/plugins/awesome-bootstrap-checkbox/awesome-bootstrap-checkbox.css" rel="stylesheet">

    <link href="css/plugins/clockpicker/clockpicker.css" rel="stylesheet">

    <link href="css/plugins/daterangepicker/daterangepicker-bs3.css" rel="stylesheet">

    <link href="css/plugins/select2/select2.min.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="active">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
             <form role="form" action="/stat" method="POST">
                    <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                        <div class="navbar-header">
                            <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i class="fa fa-bars"></i>
                            </a>
                        </div>
                        <div class="minimalize-styl-2" id="data_1">
                            <h3>From:</h3>
                        </div>
                        <div class="minimalize-styl-2" id="data_1">
                            <div class="input-group date">
                                <span class="input-group-addon"><i class="fa fa-calendar"></i></span>
                                <input name="from" required type="text" class="form-control" value="{{.From}}">
                            </div>
                        </div>
                        <div class="minimalize-styl-2" id="data_1">
                            <h3>To:</h3>
                        </div>
                        <div class="minimalize-styl-2" id="data_1">
                            <div class="input-group date">
                                <span class="input-group-addon"><i class="fa fa-calendar"></i></span>
                                <input name="to" required type="text" class="form-control" value="{{.To}}">
                            </div>
                        </div>
                        <div class="minimalize-styl-2">

                        <input type="submit" name="submit" class="btn btn-primary " value="Show Total">
                    </div>
                    </nav>


             </form>
        </div>
        <div class="wrapper wrapper-content animated fadeInRight">

            <div class="row">
                <div class="col-sm-12">
                    <div class="ibox float-e-margins">
                        <div class="ibox-title">
                            <h5>Income Graph<h5 class="pull-right">Grand Total: Rs.{{dec .Tot}}</h5></h5>
                        </div>
                        <div class="ibox-content">
                            <div id="morris-line-chart"></div>
                        </div>
                    </div>
                </div>
            </div>


        </div>
        `+footer+`
    </div>

</div>
</div>


<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>


<!-- Morris -->
<script src="js/plugins/morris/raphael-2.1.0.min.js"></script>
<script src="js/plugins/morris/morris.js"></script>

<!-- Page-Level Scripts -->
<script>
$(function() {

    Morris.Line({
        element: 'morris-line-chart',
        data: [
        {{range .Records}}
        {y: '{{ .Date}}' , a : '{{dec .Tot}}',b:'{{dec .Sale}}'},
        {{end}}
            ],
        xkey: 'y',
        ykeys: ['a','b'],
        labels: ['Total(Rs.)','Day-Sale(Rs.)'],
        hideHover: 'auto',
        resize: true,
        lineColors: ['#1ab394','#5bdfdf'],
    });

});



</script>


<!-- Chosen -->
<script src="js/plugins/chosen/chosen.jquery.js"></script>

<!-- JSKnob -->
<script src="js/plugins/jsKnob/jquery.knob.js"></script>

<!-- Input Mask-->
<script src="js/plugins/jasny/jasny-bootstrap.min.js"></script>

<!-- Data picker -->
<script src="js/plugins/datapicker/bootstrap-datepicker.js"></script>

<!-- NouSlider -->
<script src="js/plugins/nouslider/jquery.nouislider.min.js"></script>

<!-- Switchery -->
<script src="js/plugins/switchery/switchery.js"></script>

<!-- IonRangeSlider -->
<script src="js/plugins/ionRangeSlider/ion.rangeSlider.min.js"></script>

<!-- iCheck -->
<script src="js/plugins/iCheck/icheck.min.js"></script>

<!-- MENU -->
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>

<!-- Color picker -->
<script src="js/plugins/colorpicker/bootstrap-colorpicker.min.js"></script>

<!-- Clock picker -->
<script src="js/plugins/clockpicker/clockpicker.js"></script>

<!-- Image cropper -->
<script src="js/plugins/cropper/cropper.min.js"></script>

<!-- Date range use moment.js same as full calendar plugin -->
<script src="js/plugins/fullcalendar/moment.min.js"></script>

<!-- Date range picker -->
<script src="js/plugins/daterangepicker/daterangepicker.js"></script>

<!-- Select2 -->
<script src="js/plugins/select2/select2.full.min.js"></script>
<script>
        $(document).ready(function(){

            var $image = $(".image-crop > img")
            $($image).cropper({
                aspectRatio: 1.618,
                preview: ".img-preview",
                done: function(data) {
                    // Output the result data for cropping image.
                }
            });

            var $inputImage = $("#inputImage");
            if (window.FileReader) {
                $inputImage.change(function() {
                    var fileReader = new FileReader(),
                            files = this.files,
                            file;

                    if (!files.length) {
                        return;
                    }

                    file = files[0];

                    if (/^image\/\w+$/.test(file.type)) {
                        fileReader.readAsDataURL(file);
                        fileReader.onload = function () {
                            $inputImage.val("");
                            $image.cropper("reset", true).cropper("replace", this.result);
                        };
                    } else {
                        showMessage("Please choose an image file.");
                    }
                });
            } else {
                $inputImage.addClass("hide");
            }

            $("#download").click(function() {
                window.open($image.cropper("getDataURL"));
            });

            $("#zoomIn").click(function() {
                $image.cropper("zoom", 0.1);
            });

            $("#zoomOut").click(function() {
                $image.cropper("zoom", -0.1);
            });

            $("#rotateLeft").click(function() {
                $image.cropper("rotate", 45);
            });

            $("#rotateRight").click(function() {
                $image.cropper("rotate", -45);
            });

            $("#setDrag").click(function() {
                $image.cropper("setDragMode", "crop");
            });

            $('#data_1 .input-group.date').datepicker({
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                calendarWeeks: true,
                autoclose: true
            });

            $('#data_2 .input-group.date').datepicker({
                startView: 1,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                format: "dd/mm/yyyy"
            });

            $('#data_3 .input-group.date').datepicker({
                startView: 2,
                todayBtn: "linked",
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            $('#data_4 .input-group.date').datepicker({
                minViewMode: 1,
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true,
                todayHighlight: true
            });

            $('#data_5 .input-daterange').datepicker({
                keyboardNavigation: false,
                forceParse: false,
                autoclose: true
            });

            var elem = document.querySelector('.js-switch');
            var switchery = new Switchery(elem, { color: '#1AB394' });

            var elem_2 = document.querySelector('.js-switch_2');
            var switchery_2 = new Switchery(elem_2, { color: '#ED5565' });

            var elem_3 = document.querySelector('.js-switch_3');
            var switchery_3 = new Switchery(elem_3, { color: '#1AB394' });

            $('.i-checks').iCheck({
                checkboxClass: 'icheckbox_square-green',
                radioClass: 'iradio_square-green'
            });

            $('.demo1').colorpicker();

            var divStyle = $('.back-change')[0].style;
            $('#demo_apidemo').colorpicker({
                color: divStyle.backgroundColor
            }).on('changeColor', function(ev) {
                        divStyle.backgroundColor = ev.color.toHex();
                    });

            $('.clockpicker').clockpicker();

            $('input[name="daterange"]').daterangepicker();

            $('#reportrange span').html(moment().subtract(29, 'days').format('MMMM D, YYYY') + ' - ' + moment().format('MMMM D, YYYY'));

            $('#reportrange').daterangepicker({
                format: 'MM/DD/YYYY',
                startDate: moment().subtract(29, 'days'),
                endDate: moment(),
                minDate: '01/01/2012',
                maxDate: '12/31/2015',
                dateLimit: { days: 60 },
                showDropdowns: true,
                showWeekNumbers: true,
                timePicker: false,
                timePickerIncrement: 1,
                timePicker12Hour: true,
                ranges: {
                    'Today': [moment(), moment()],
                    'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
                    'Last 7 Days': [moment().subtract(6, 'days'), moment()],
                    'Last 30 Days': [moment().subtract(29, 'days'), moment()],
                    'This Month': [moment().startOf('month'), moment().endOf('month')],
                    'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
                },
                opens: 'right',
                drops: 'down',
                buttonClasses: ['btn', 'btn-sm'],
                applyClass: 'btn-primary',
                cancelClass: 'btn-default',
                separator: ' to ',
                locale: {
                    applyLabel: 'Submit',
                    cancelLabel: 'Cancel',
                    fromLabel: 'From',
                    toLabel: 'To',
                    customRangeLabel: 'Custom',
                    daysOfWeek: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr','Sa'],
                    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
                    firstDay: 1
                }
            }, function(start, end, label) {
                console.log(start.toISOString(), end.toISOString(), label);
                $('#reportrange span').html(start.format('MMMM D, YYYY') + ' - ' + end.format('MMMM D, YYYY'));
            });

            $(".select2_demo_1").select2();
            $(".select2_demo_2").select2();
            $(".select2_demo_3").select2({
                placeholder: "Select a state",
                allowClear: true
            });


        });
        var config = {
                '.chosen-select'           : {},
                '.chosen-select-deselect'  : {allow_single_deselect:true},
                '.chosen-select-no-single' : {disable_search_threshold:10},
                '.chosen-select-no-results': {no_results_text:'Oops, nothing found!'},
                '.chosen-select-width'     : {width:"95%"}
                }
            for (var selector in config) {
                $(selector).chosen(config[selector]);
            }

        $("#ionrange_1").ionRangeSlider({
            min: 0,
            max: 5000,
            type: 'double',
            prefix: "$",
            maxPostfix: "+",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_2").ionRangeSlider({
            min: 0,
            max: 10,
            type: 'single',
            step: 0.1,
            postfix: " carats",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_3").ionRangeSlider({
            min: -50,
            max: 50,
            from: 0,
            postfix: "°",
            prettify: false,
            hasGrid: true
        });

        $("#ionrange_4").ionRangeSlider({
            values: [
                "January", "February", "March",
                "April", "May", "June",
                "July", "August", "September",
                "October", "November", "December"
            ],
            type: 'single',
            hasGrid: true
        });

        $("#ionrange_5").ionRangeSlider({
            min: 10000,
            max: 100000,
            step: 100,
            postfix: " km",
            from: 55000,
            hideMinMax: true,
            hideFromTo: false
        });

        $(".dial").knob();

        $("#basic_slider").noUiSlider({
            start: 40,
            behaviour: 'tap',
            connect: 'upper',
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#range_slider").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });

        $("#drag-fixed").noUiSlider({
            start: [ 40, 60 ],
            behaviour: 'drag-fixed',
            connect: true,
            range: {
                'min':  20,
                'max':  80
            }
        });


</script>
</body>
</html>

	`
	files["unload"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">

    <!-- FooTable -->
    <link href="css/plugins/footable/footable.core.css" rel="stylesheet">
</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="active">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                <div class="navbar-header">
                    <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i class="fa fa-bars"></i>
                    </a>
                </div>

            </nav>
        </div>

        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-12">
                    <div class="ibox">
                        <div class="ibox-content">
                            <h2>Vehicles</h2>
                            <div class="input-group">
                                <input type="text" class="input form-control"
                                       id="filter"
                                       placeholder="Search in Unloadinig">
                                </span>
                            </div>
                            <div class="clients-list">
                                <ul class="nav nav-tabs">
                                    {{range .Vans}}
                                    <li {{if .Active}}class="active" {{end}}><a data-toggle="tab" href="#van{{.Id}}"><i
                                            class="fa fa-bus"></i>
                                        {{.Des}}</a></li>
                                    {{end}}
                                </ul>
                                <div class="tab-content">
                                    {{range .Vans}}
                                    <div id="van{{.Id}}" class="tab-pane {{if .Active}}active{{end}}">
                                        <div class="full-height-scroll">
                                            <div class="col-lg-12">
                                                <div class="ibox float-e-margins">
                                                    <div class="ibox-content">
                                                        <table class="footable table table-stripped" data-page-size="8"
                                                               data-filter=#filter>
                                                            <thead>
                                                            <tr>
                                                                <th>Date</th>
                                                                <th>Product</th>
                                                                <th>Qty.</th>
                                                            </tr>
                                                            </thead>
                                                            <tbody>
                                                            {{range .Unloads}}
                                                            <tr class="gradeU">
                                                                <td>{{.Dte}}</td>
                                                                <td>{{.P_des}}</td>
                                                                <td>{{.Qty}}</td>
                                                            </tr>
                                                            {{end}}
                                                            </tbody>
                                                            <tfoot>
                                                            <tr>
                                                                <td colspan="5">
                                                                    <ul class="pagination pull-right"></ul>
                                                                </td>
                                                            </tr>
                                                            </tfoot>
                                                        </table>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>


                                    </div>
                                    {{end}}
                                </div>

                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
`+footer+`
    </div>
</div>


<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>


<!-- FooTable -->
<script src="js/plugins/footable/footable.all.min.js"></script>


<!-- Page-Level Scripts -->
<script>
        $(document).ready(function() {

            $('.footable').footable();
            $('.footable2').footable();

        });








</script>
</body>
</html>

	`
	files["vendors"] = `
	<!DOCTYPE html>
<html>
<head>

    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>{{.Title}}</title>

    <link href="css/bootstrap.min.css" rel="stylesheet">
    <link href="font-awesome/css/font-awesome.css" rel="stylesheet">

    <!-- Toastr style -->
    <link href="css/plugins/toastr/toastr.min.css" rel="stylesheet">

    <link href="css/animate.css" rel="stylesheet">
    <link href="css/style.css" rel="stylesheet">


</head>

<body>

<div id="wrapper">

    <nav class="navbar-default navbar-static-side" role="navigation">
        <div class="sidebar-collapse">
            <ul class="nav metismenu" id="side-menu">
                <li class="">
                    <a href="stat"><i class="fa fa-line-chart"></i> <span class="nav-label">Stats</span></a>
                </li>
                <li class="">
                    <a href="payment"><i class="fa fa-money"></i> <span class="nav-label">Payment</span></a>
                </li>
                <li class="">
                    <a href="load"><i class="fa fa-download"></i> <span class="nav-label">Loading</span></a>
                </li>
                <li class="">
                    <a href="unload"><i class="fa fa-upload"></i> <span class="nav-label">Unloading</span></a>
                </li>
                <li class="">
                    <a href="delivery"><i class="fa fa-bus"></i> <span class="nav-label">Delivery</span></a>
                </li>
                <li class="">
                    <a href="invoice"><i class="fa fa-usd"></i> <span class="nav-label">Invoice</span></a>
                </li>
                <li class="">
                    <a href="grn"><i class="fa fa-gbp"></i> <span class="nav-label">GRN</span></a>
                </li>
                <li class="">
                    <a href="products"><i class="fa fa-cubes"></i> <span class="nav-label">Products</span></a>
                </li>
                <li class="">
                    <a href="customers"><i class="fa fa-slideshare"></i> <span class="nav-label">Customers</span></a>
                </li>
                <li class="active">
                    <a href="vendors"><i class="fa fa-users"></i> <span class="nav-label">Vendors</span></a>
                </li>
                <li class="">
                    <a href="home"><i class="fa fa-home"></i> <span class="nav-label">Home</span></a>
                </li>
            </ul>

        </div>
    </nav>

    <div id="page-wrapper" class="gray-bg">
        <div class="row border-bottom">
            <nav class="navbar navbar-static-top" role="navigation" style="margin-bottom: 0">
                <div class="navbar-header">
                    <a class="navbar-minimalize minimalize-styl-2 btn btn-primary " href="#"><i class="fa fa-bars"></i>
                    </a>
                </div>
                <div class="">
                    <a data-toggle="modal" href="#Addnew" class="btn btn-primary minimalize-styl-2">Add new</a>
                </div>

            </nav>
        </div>


        <div id="Addnew" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form role="form" action="/vendors" method="POST">
                                    <div class="form-group"><label>ID</label>
                                        <input name="id" readonly required type="text" placeholder="ID"
                                               class="form-control" value="{{.Nid}}">
                                    </div>
                                    <div class="form-group"><label>Name</label>
                                        <input name="name" required type="text" placeholder="Name"
                                               class="form-control" value="">
                                    </div>
                                    <div class="form-group"><label>Phone</label>
                                        <input name="phn" type="text" placeholder="Phone"
                                               class="form-control" value="">
                                    </div>
                                    <div class="form-group"><label>Address</label>
                                        <input name="ad" type="text" placeholder="Address"
                                               class="form-control" value="">
                                    </div>

                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs"
                                                       value="Add"></strong>
                                        <strong><input type="reset" class="btn btn-sm btn-warning pull-right m-t-n-xs"
                                                       value="Reset"></strong>
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        {{range .Vendors}}
        <div id="c{{.Id}}" class="modal fade" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-body">
                        <div class="row">
                            <div class=""><h3 class="m-t-none m-b">Details</h3>

                                <form role="form" action="/vendors" method="POST">
                                    <div class="form-group"><label>ID</label>
                                        <input name="id" readonly required type="text" placeholder="ID"
                                               class="form-control" value="{{.Id}}">
                                    </div>
                                    <div class="form-group"><label>Name</label>
                                        <input name="name" required type="text" placeholder="Name"
                                               class="form-control" value="{{.Name}}">
                                    </div>
                                    <div class="form-group"><label>Phone</label>
                                        <input name="phn" type="text" placeholder="Phone"
                                               class="form-control" value="{{.Phn}}">
                                    </div>
                                    <div class="form-group"><label>Address</label>
                                        <input name="ad" type="text" placeholder="Address"
                                               class="form-control" value="{{.Ad}}">
                                    </div>

                                    <div>
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-primary pull-right m-t-n-xs" value="Save"></strong>
                                        {{if .DeleteBTN}}
                                        <strong><input name="submit" type="submit"
                                                       class="btn btn-sm btn-danger pull-right m-t-n-xs" value="Delete"></strong>
                                        {{end}}
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        {{end}}


        <div class="wrapper wrapper-content  animated fadeInRight">
            <div class="row">
                <div class="col-sm-7">
                    <div class="ibox">
                        <div class="ibox-content">
                            <div class="clients-list">
                                <div class="tab-content">
                                    <div class="tab-pane active">
                                        <div class="full-height-scroll">
                                            <div class="table-responsive">
                                                <table class="table table-striped table-hover dataTables-example">
                                                    <thead>
                                                    <tr>
                                                        <th>Name</th>
                                                        <th>Phone</th>
                                                        <th>Total</th>
                                                        <th>Option</th>
                                                    </tr>
                                                    </thead>
                                                    <tbody>
                                                    {{range .Vendors}}
                                                    <tr>
                                                        <td><a data-toggle="tab" href="#cus{{.Id}}" class="client-link">
                                                            {{.Name}}
                                                        </a></td>
                                                        <td>{{.Phn}}</td>
                                                        <td>{{dec .Dne}}</td>
                                                        <td><a data-toggle="modal"
                                                               class="btn btn-info btn-sm btn-outline" href="#c{{.Id}}">View
                                                            /
                                                            Edit</a>
                                                        </td>
                                                    </tr>
                                                    {{end}}
                                                    </tbody>
                                                </table>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-sm-5">
                    <div class="ibox ">

                        <div class="ibox-content">
                            <div class="tab-content">
                                {{range .Vendors}}
                                <div id="cus{{.Id}}" class="tab-pane {{if .Active}} active {{end}}">
                                    <div class="m-b-lg">
                                        <h2>{{.Name}}</h2>
                                        <br>
                                        <p>
                                            {{.Phn}}
                                            <br>
                                            {{.Ad}}
                                            <br>
                                            Total payment: {{dec .Dne}}
                                        </p>
                                    </div>
                                    <div class="client-detail">
                                        <div class="full-height-scroll">
                                            <strong>Timeline activity</strong>

                                            <div id="vertical-timeline" class="vertical-container dark-timeline">
                                                {{range .Grns}}
                                                <div class="vertical-timeline-block">
                                                    <div class="vertical-timeline-icon navy-bg">
                                                        <i class="fa fa-gbp"></i>
                                                    </div>
                                                    <div class="vertical-timeline-content">

                                                        <p>GRN. No.:{{.G_no}}
                                                            Total:{{dec .Grnd_tot}}
                                                            <button type="button" class="btn btn-info btn-xs"
                                                                    data-toggle="modal" data-target="#i{{.Id}}">
                                                                GRN
                                                            </button>
                                                        </p>
                                                        <span class="vertical-date small text-muted"> Date: {{.Dte}} </span>
                                                    </div>
                                                </div>

                                                {{end}}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                {{end}}

                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

`+footer+`

    </div>
</div>

{{range .Vendors}}
{{range .Grns}}
<div class="modal inmodal fade" id="i{{.Id}}" tabindex="-1" role="dialog" aria-hidden="true">
    <div class="modal-dialog modal-md">
        <div class="modal-content">
            <div class="modal-header">
                <button type="button" class="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span
                        class="sr-only">Close</span></button>
                <h4 class="modal-title">GRN. No.:{{.G_no}} </h4>
                <h5>Date:{{.Dte}}</h5>
            </div>
            <div class="modal-body">
                <table class="table table-striped table-bordered">
                    <thead>
                    <tr>
                        <th>Description</th>
                        <th>Qty</th>
                        <th>U.Price</th>
                        <th>Total</th>
                    </tr>
                    </thead>
                    <tbody>
                    {{range .Records}}
                    <tr>
                        <td>{{.P_des}}</td>
                        <td>{{.Qty}}</td>
                        <td>{{dec .B_p}}</td>
                        <td>{{dec .Tot}}</td>
                    </tr>
                    {{end}}
                    <tr>
                        <td></td>
                        <td></td>
                        <td></td>
                        <td></td>
                    </tr>
                    <tr>
                        <td><strong>Sub Total</strong></td>
                        <td></td>
                        <td></td>
                        <td>{{dec .Sub_tot}}</td>
                    </tr>
                    <tr>
                        <td><strong>Vat</strong></td>
                        <td></td>
                        <td></td>
                        <td>{{dec .Vat}}</td>
                    </tr>
                    <tr>
                        <td><strong>Grand Total</strong></td>
                        <td></td>
                        <td></td>
                        <td>{{dec .Grnd_tot}}</td>
                    </tr>
                    </tbody>
                </table>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-white btn-sm" data-dismiss="modal">Close</button>
                <a href="editGRN?id={{.Id}}" type="button" class="btn btn-warning btn-sm">Edit/View</a>
            </div>
        </div>
    </div>
</div>
{{end}}
{{end}}

<!-- Mainly scripts -->
<script src="js/jquery-2.1.1.js"></script>
<script src="js/bootstrap.min.js"></script>
<script src="js/plugins/metisMenu/jquery.metisMenu.js"></script>
<script src="js/plugins/slimscroll/jquery.slimscroll.min.js"></script>

<!-- Custom and plugin javascript -->
<script src="js/inspinia.js"></script>
<script src="js/plugins/pace/pace.min.js"></script>

</body>
</html>

	`
}
