package main

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
	StartRest      int
	RestPort       int
	RestHost       string
	StartChecks    int
}

type restHandler struct{}

type CompInvRow struct {
	ItemNO int
	Serial string
	Model  string
	Type   string
	Owner  string
}

type ServInvRow struct {
	ServiceID int
	Address   string
	Port      string
	Protocol  string
	SDesc     string
	Serial    string
}

type ServTimeRow struct {
	Entry     string
	Latency   int
	Status    rune
	ServiceID int
}

type CompInvResp struct {
	Devices []CompInvRow
	Code    int
	Message string
}

type ServInvResp struct {
	Service  ServInvRow
	Services []ServInvRow
	Code     int
	Message  string
}

type ServTimeResp struct {
	Entries []ServTimeRow
	Code    int
	Message string
}
