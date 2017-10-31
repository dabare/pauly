package main

import (
	"net/http"
	"io/ioutil"
	"log"
	"html/template"
	"database/sql"
	//_ "github.com/mattn/go-sqlite3"
	_ "github.com/mxk/go-sqlite/sqlite3"
	"strconv"
	"strings"
	"time"
	"os"
)

var mysql_user string = "root"
var mysql_pass string = "aelo"
var mysql_db string = "mandy"

var errDefault = 0
var errMysqlDBname = 1
var errDBquery = 2

var dbName = "mandy"

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

func debugMSG(msg string) {
	println(msg)
}

func readFile(path string) string {
	s, _ := ioutil.ReadFile("./core/" + path)
	return string(s)
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
	Tot  int64
}

func getIncome(from string, to string) [] income {
	tmp2 := []income{}
	rows := getResultDB("select sum(inv_reg.s_p*inv_reg.qty+inv.vat) as tot , inv.dte from inv_reg join inv on inv_reg.i_id=inv.id where inv.dte between '" + from + "' and '" + to + "' group by inv.dte order by inv.dte asc")

	for rows.Next() {
		tmp := income{}

		var tot sql.NullInt64
		var dte sql.RawBytes

		err := rows.Scan(&tot, &dte)
		checkErr(err, errDBquery)

		tmp.Tot = tot.Int64
		tmp.Date = string(dte)

		tmp2 = append(tmp2, tmp)
	}
	rows.Close()
	return tmp2
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
		From    string
		To      string
	}
	now := time.Now()

	now.Format("01/02/2006")

	//select sum(inv_reg.s_p*inv_reg.qty+inv.vat) as tot , inv.dte from inv_reg join inv on inv_reg.i_id=inv.id where inv.dte between '2017-09-07' and '2017-12-14' group by inv.dte order by inv.dte asc;
	from := now.Format("01") + "/1/" + now.Format("2006")
	to := now.Format("01/02/2006")

	if (r.Form.Get("from")!="") {
		from = r.Form.Get("from")
	}
	if (r.Form.Get("to")!="") {
		to = r.Form.Get("to")
	}

	//m,d,y
	s := strings.Split(from, "/")
	sqlFrom := s[2] + "-" + s[0] + "-" + s[1]

	s = strings.Split(to, "/")
	sqlTo := s[2] + "-" + s[0] + "-" + s[1]

	results := data{dbName, getIncome(sqlFrom, sqlTo), from, to}
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

		if len(invs) > 0 {
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
	err := http.ListenAndServe("localhost:8008", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
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
	startService()

}
