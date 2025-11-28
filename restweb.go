package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func setupREST() {
	out.Println("Begin setupREST()")
	router := mux.NewRouter()
	rest := restHandler{}

	router.HandleFunc("/device/all", rest.getInventory)
	router.HandleFunc("/service/{serviceID}/uptime", rest.getServUptime)
	router.HandleFunc("/service/{serviceID}", rest.getServInfo)
	router.HandleFunc("/service", rest.getServices)

	out.Println("Pass REST off to net/http")
	addr := fmt.Sprintf("%s:%d", activeConfig.RestHost, activeConfig.RestPort)
	http.ListenAndServe(addr, router)
}

func (h *restHandler) getServices(w http.ResponseWriter, r *http.Request) {
	out.Println("Begin getServices()")
	profiler := time.Now()
	var resp []ServInvRow

	st, err := db.Prepare(fmt.Sprintf(
		`SELECT B.SERVICE_ID
      		   ,AVG(A.LATENCY) AS AVG_LATENCY 
      		   ,B.SDESC 
			   ,B.ADDRESS 
		       ,B.PORT 
		       ,B.PROTOCOL 
			   ,B.SERIAL
		FROM %s.%s A INNER JOIN %s.%s B 
		ON A.SERVICE_ID = B.SERVICE_ID 
		GROUP BY B.SERVICE_ID, B.SDESC, B.ADDRESS, B.PORT, B.PROTOCOL, B.SERIAL; `,
		activeConfig.TargetSchema, activeConfig.UptimeTable, activeConfig.TargetSchema, activeConfig.ServiceTable))
	if err != nil {
		out.Printf("Unable to prepare statement: %+v", err)
		return
	}
	rows, err := st.Query()
	if err != nil {
		out.Printf("Error returned from st.Query(): %+v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rServiceID int
		var rSerial, rDesc, rAddr, rPort, rProto string
		var rLatency float32
		err = rows.Scan(&rServiceID, &rLatency, &rDesc, &rAddr, &rPort, &rProto, &rSerial)
		if err != nil {
			out.Printf("Error returned from rows.Scan(): %+v", err)
		}
		resp = append(resp, ServInvRow{ServiceID: rServiceID, Serial: rSerial, SDesc: rDesc, Address: rAddr, Latency: rLatency})
	}
	structResp := ServInvResp{Services: resp, Code: 200, Message: "OK"}
	jsonResp, err := json.Marshal(structResp)
	if err != nil {
		out.Printf("Error returned from json.Marshal(): %+v", err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResp)
	}
	out.Printf("getInventory finished in %+v", time.Since(profiler))

}

func (h *restHandler) getServUptime(w http.ResponseWriter, r *http.Request) {
	out.Println("Begin getServUptime()")
	profiler := time.Now()
	var resp []ServTimeRow
	vars := mux.Vars(r)
	serviceID := vars["serviceID"]

	st, err := db.Prepare(fmt.Sprintf("select ENTRY,STATUS,LATENCY from %s.%s WHERE SERVICE_ID = %s ORDER BY ENTRY DESC LIMIT 128",
		activeConfig.TargetSchema, activeConfig.UptimeTable, serviceID))
	if err != nil {
		out.Printf("Unable to prepare statement: %+v", err)
		return
	}
	rows, err := st.Query()
	if err != nil {
		out.Printf("Error returned from st.Query(): %+v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rStatus []uint8
		var rLatency float32
		var rEntry string
		err = rows.Scan(&rEntry, &rStatus, &rLatency)
		if err != nil {
			out.Printf("Error returned from rows.Scan(): %+v", err)
		}
		if err != nil {
			out.Printf("Error returned from st.Query(): %+v", err)
			return
		}
		parsedServiceID, err := strconv.Atoi(serviceID)
		if err != nil {
			out.Printf("Error converting serviceID to int: %+v", err)
			return
		}
		resp = append(resp, ServTimeRow{Entry: rEntry, Status: rune(rStatus[0]), Latency: rLatency, ServiceID: parsedServiceID})
	}

	structResp := ServTimeResp{Entries: resp, Code: 200, Message: "OK"}
	jsonResp, err := json.Marshal(structResp)
	if err != nil {
		out.Printf("Error returned from json.Marshal(): %+v", err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResp)
	}
	out.Printf("getInventory finished in %+v", time.Since(profiler))
}

func (h *restHandler) getServInfo(w http.ResponseWriter, r *http.Request) {
	out.Println("Begin getServInfo()")
	profiler := time.Now()
	var resp ServInvRow
	vars := mux.Vars(r)
	serviceID := vars["serviceID"]

	st, err := db.Prepare(fmt.Sprintf("select SERIAL,SDESC,ADDRESS,PORT,PROTOCOL from %s.%s WHERE SERVICE_ID = %s",
		activeConfig.TargetSchema, activeConfig.ServiceTable, serviceID))
	if err != nil {
		out.Printf("Unable to prepare statement: %+v", err)
		return
	}
	rows, err := st.Query()
	if err != nil {
		out.Printf("Error returned from st.Query(): %+v", err)
		return
	}
	defer rows.Close()
	var qSerial, qDesc, qAddr, qPort, qProto string
	rows.Next()
	err = rows.Scan(&qSerial, &qDesc, &qAddr, &qPort, &qProto)
	if err != nil {
		out.Printf("Error returned from rows.Scan(): %+v", err)
	}
	parsedServiceID, err := strconv.Atoi(serviceID)
	if err != nil {
		out.Printf("Error returned from strconv.Atoi(): %+v", err)
	}
	resp = ServInvRow{Serial: qSerial, SDesc: qDesc, Address: qAddr, Port: qPort, Protocol: qProto, ServiceID: parsedServiceID}
	structResp := ServInvResp{Service: resp, Code: 200, Message: "OK"}
	jsonResp, err := json.Marshal(structResp)
	if err != nil {
		out.Printf("Error returned from json.Marshal(): %+v", err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResp)
	}
	out.Printf("getInventory finished in %+v", time.Since(profiler))
}

func (h *restHandler) getInventory(w http.ResponseWriter, r *http.Request) {
	out.Println("Begin getInventory()")
	profiler := time.Now()

	var resp []CompInvRow
	st, err := db.Prepare(fmt.Sprintf("select ItemNO,SERIAL,MODEL,TYPE,OWNER from %s.%s",
		activeConfig.TargetSchema, activeConfig.InventoryTable))
	if err != nil {
		out.Printf("Unable to prepare statement: %+v", err)
		return
	}
	rows, err := st.Query()
	if err != nil {
		out.Printf("Error returned from st.Query(): %+v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var qSerial, qModel, qType, qOwner string
		var qItemNo int
		err = rows.Scan(&qItemNo, &qSerial, &qModel, &qType, &qOwner)
		if err != nil {
			out.Printf("Error returned from rows.Scan(): %+v", err)
		}
		resp = append(resp, CompInvRow{ItemNO: qItemNo, Serial: qSerial, Model: qModel, Type: qType, Owner: qOwner})
	}
	structResp := CompInvResp{Devices: resp, Code: 200, Message: "OK"}
	jsonResp, err := json.Marshal(structResp)
	if err != nil {
		out.Printf("Error returned from json.Marshal(): %+v", err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResp)
	}
	out.Printf("getInventory finished in %+v", time.Since(profiler))

}
