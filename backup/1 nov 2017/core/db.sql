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
