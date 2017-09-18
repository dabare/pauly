DROP database mandy;

CREATE DATABASE mandy;

USE mandy

CREATE TABLE cus (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    phn VARCHAR(255),
    ad VARCHAR(255)
);

insert into cus values (0,"madhav","sadas","adfsgad");
insert into cus values (1,"dfg","sadas","adfsad");
insert into cus values (2,"madhdfgav","sadfgdas","adfgsad");
insert into cus values (3,"dfg","saddfgas","adfsgad");

CREATE TABLE cus_pay (
    id INT PRIMARY KEY,
     dte DATE,
     i_id INT,
     tot INT
);
insert into cus_pay values(0,"12-12-12",0,2);
insert into cus_pay values(1,"12-12-12",0,56);

insert into cus_pay values(2,"12-12-12",1,232);

CREATE TABLE ven (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    phn VARCHAR(255),
    ad VARCHAR(255)
);

CREATE TABLE pro (
    id INT PRIMARY KEY,
    des VARCHAR(255),
    s_p INT,
    b_p INT
);
INSERT INTO pro VALUES(0,'Pen',2,1);
INSERT INTO pro VALUES(1,'Pencile',3,2);
INSERT INTO pro VALUES(2,'Apple',5,4);

CREATE TABLE van (
    id INT PRIMARY KEY,
    des VARCHAR(255)
);

CREATE TABLE ldng (
id INT PRIMARY KEY,
    v_id INT,
    p_id INT,
    qty INT,
    dte DATE
);

CREATE TABLE u_ldng (
id INT PRIMARY KEY,
    v_id INT,
    p_id INT,
    qty INT,
    dte DATE
);

CREATE TABLE stk_main (
id INT PRIMARY KEY,
    p_id INT,
    qty INT
);

CREATE TABLE grn (
    id INT PRIMARY KEY,
    v_id INT,
    g_no VARCHAR(255),
    vat INT,
    dte DATE
);
insert into grn values(0,0,"ds",23,"1212-12-12");
insert into grn values(1,0,"ds",23,"1212-12-13");
insert into grn values(2,0,"ds",12,"1212-12-14");
Insert into grn values (3,2,"ty",2,"12-12-12");

CREATE TABLE grn_reg (
    id INT PRIMARY KEY,
    g_id INT,
    p_id INT,
    b_p INT,
    qty INT
);

insert into grn_reg values (0,0,0,90,12);
insert into grn_reg values (1,0,1,900,12);
insert into grn_reg values (2,1,0,90,12);

CREATE TABLE inv (
    id INT PRIMARY KEY,
    c_id INT,
    i_no VARCHAR(255),
    po_no VARCHAR(255),
    vat INT,
    dte DATE
);

insert into inv values(0,0,"ds","sdf",23,"1212-12-12");
insert into inv values(1,0,"ds","sdf",23,"1212-12-13");
insert into inv values(2,0,"ds","sdf",23,"1212-12-14");

CREATE TABLE inv_reg (
    id INT PRIMARY KEY,
    i_id INT,
    p_id INT,
    b_p INT,
    s_p INT,
    qty INT
);

insert into inv_reg values(0,0,0,12,12,12);
insert into inv_reg values(1,0,1,12,12,12);

insert into inv_reg values(2,1,0,12,12,12);
insert into inv_reg values(3,1,1,12,12,12);
insert into inv_reg values(4,1,2,12,12,12);