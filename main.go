package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/ibmdb/go_ibm_db"
)

type CompInvConfig struct {
	HostName       string
	Database       string
	Port           int
	UID            string
	PWD            string
	Heartbeat      int
	InventoryTable string
	ServiceTable   string
	UptimeTable    string
	TargetSchema   string
}

var (
	activeConfig CompInvConfig
	db           *sql.DB
)

func main() {
	f := "CompInvZ.toml"
	_, err := toml.DecodeFile(f, &activeConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	con := fmt.Sprintf("HOSTNAME=%s;DATABASE=%s;PORT=%d;UID=%s;PWD=%s",
		activeConfig.HostName, activeConfig.Database, activeConfig.Port,
		activeConfig.UID, activeConfig.PWD)
	db, err = sql.Open("go_ibm_db", con)
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	for {
		err = testAllServices(db)
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Duration(activeConfig.Heartbeat) * time.Second)
	}
}
func testService(serviceID int, serviceAddress string, servicePort string, serviceProto string) {
	timeout := 8 * time.Second
	active := 'D'
	begin := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(serviceAddress, servicePort), timeout)
	if err != nil {
		active = 'D'
	}
	if conn != nil {
		defer conn.Close()
		active = 'U'
	}
	lat := time.Since(begin)
	st, err := db.Prepare(fmt.Sprintf("INSERT into %s.%s(LATENCY,STATUS,SERVICE_ID) values(%d,'%c',%d)",
		activeConfig.TargetSchema, activeConfig.UptimeTable, int(lat.Milliseconds()), active, serviceID))
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = st.Query()
	fmt.Println(err)
}
func testAllServices(db *sql.DB) error {
	st, err := db.Prepare(fmt.Sprintf("select SERVICE_ID,ADDRESS,PORT,PROTOCOL from %s.%s",
		activeConfig.TargetSchema, activeConfig.ServiceTable))
	if err != nil {
		return err
	}
	rows, err := st.Query()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var sAddr, sProto, sPort string
		var sID int
		err = rows.Scan(&sID, &sAddr, &sPort, &sProto)
		if err != nil {
			return err
		}
		go testService(sID, sAddr, sPort, sProto)
	}

	return nil
}
