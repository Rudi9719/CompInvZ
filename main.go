package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/ibmdb/go_ibm_db"
)

var (
	activeConfig CompInvConfig
	db           *sql.DB
	out          *log.Logger
)

func main() {
	out = log.Default()
	out.Println("Opening Config file CompInvZ.toml")
	f := "CompInvZ.toml"
	_, err := toml.DecodeFile(f, &activeConfig)
	if err != nil {
		out.Fatal(err)
	}
	out.Println("Config active, opening Db2 connection")
	con := fmt.Sprintf("HOSTNAME=%s;DATABASE=%s;PORT=%d;UID=%s;PWD=%s",
		activeConfig.HostName, activeConfig.Database, activeConfig.Port,
		activeConfig.UID, activeConfig.PWD)
	db, err = sql.Open("go_ibm_db", con)
	if err != nil {
		out.Fatalf("Unable to open Db2 connection: %+v", err)
	}
	defer db.Close()
	out.Println("Db2 connection ready")
	if activeConfig.StartRest == 1 {
		out.Println("Starting REST service")
		go setupREST()
	}
	if activeConfig.StartChecks == 1 {
		out.Println("Starting service checks")
		for {
			err = testAllServices(db)
			if err != nil {
				out.Printf("Error returned from testAllServices(): %+v", err)
			}
			time.Sleep(time.Duration(activeConfig.Heartbeat) * time.Second)
		}
	} else if activeConfig.StartRest == 1 {
		for {
			out.Println("Keeping main thread alive for REST service")
			time.Sleep(10 * time.Minute)
		}
	}

}
func testService(serviceID int, serviceAddress string, servicePort string, serviceProto string) {
	timeout := 8 * time.Second
	active := 'D'
	begin := time.Now()
	conn, err := net.DialTimeout(serviceProto, net.JoinHostPort(serviceAddress, servicePort), timeout)
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
		out.Printf("Unable to insert service entry: %+v", err)
		return
	}
	_, err = st.Query()
	out.Printf("%+v", err)
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
