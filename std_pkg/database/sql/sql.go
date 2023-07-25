package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type City struct {
	ID int32
	Name string
	CountryCode string
	District string
	Population int
}

func initDB() *sql.DB {
	dsn := "test:123456@tcp(localhost:3306)/world"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("open db err: %v", err))
	}

	err = db.Ping()
	if  err != nil {
		panic(fmt.Sprintf("ping err: %v", err))
	}

	return db
}

func main() {
	db := initDB()

	/*
		Insert
	*/

	//ret, err := db.Exec("insert into city(Name, CountryCode, Population) values(?, ?, ?)", "Beijing", "CHN", 1000000)
	//if err != nil {
	//	fmt.Printf("Insert error: %v", err)
	//	return
	//}
	//
	//id, err := ret.LastInsertId()
	//if err != nil {
	//	fmt.Printf("Get ID error: %v", err)
	//	return
	//}
	//
	//fmt.Printf("ID is %d\n", id)

	/*
		Select
	*/

	// 1. 单行查询
	//row := db.QueryRow("select * from city where CountryCode = ? Order by id desc", "CHN")
	//var city City
	//// 参数数量 & 顺序必须跟表结构一致 ！！！
	//// 单行查询，必须要调用 Scan 方法，才能释放链接
	//err := row.Scan(&city.ID, &city.Name, &city.CountryCode, &city.District, &city.Population)
	//if err != nil {
	//	fmt.Printf("query error: %v", err)
	//	return
	//}
	//fmt.Printf("Result is %+v\n", city)

	// 2. 多行查询
	rows, err := db.Query("select * from city where CountryCode = ? order by id desc", "CHN")
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		return
	}

	// 多行查询必须通过调用 Close() 来释放底层 TCP 链接。
	defer rows.Close()

	for rows.Next() {
		var city City
		err = rows.Scan(&city.ID, &city.Name, &city.CountryCode, &city.District, &city.Population)
		if err != nil {
			fmt.Printf("Scan error: %v\n", err)
			return
		}
		fmt.Printf("Row: %#v\n", city)
	}

	/*
		***************** UPDATE ****************
	*/

	//ret, err := db.Exec("update city set population = 10000000 where district = 'peking'")
	//if err != nil {
	//	fmt.Printf("Update error: %v\n", err)
	//	return
	//}
	//
	//rowCount, err := ret.RowsAffected()
	//if err != nil {
	//	fmt.Printf("RowsAffected error: %v\n", err)
	//	return
	//}
	//
	//fmt.Printf("RowsAffected: %d\n", rowCount)

	/*
		***************** DELETE *******************
	*/

	//ret, err := db.Exec("delete from city where id = ?", 4081)
	//if err != nil {
	//	fmt.Printf("Delete error: %v\n", err)
	//	return
	//}
	//
	//rowCount, err := ret.RowsAffected()
	//if err != nil {
	//	fmt.Printf("RowsAffected error: %v\n", err)
	//	return
	//}
	//
	//fmt.Printf("RowsAffected %d\n", rowCount)


	/*
		************ TRANSACTION *************
	*/

	//tx, err := db.Begin()
	//if err != nil {
	//	fmt.Printf("tx begin error: %v\n", err)
	//	return
	//}
	//
	//ret, err := tx.Exec("insert into city(name, countrycode, district, population) values(?, ?, ?, ?)", "Beijing", "CHN", "Beijing", 10000000)
	//if err != nil {
	//	tx.Rollback()
	//	fmt.Printf("insert error: %v, rollback!", err)
	//	return
	//}
	//tx.Commit()
	//id, _ := ret.LastInsertId()
	//fmt.Printf("new id: %d\n", id)
}