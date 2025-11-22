package process

// CommonPorts is a list of commonly used ports for scanning.
var CommonPorts = []int{
	21, 22, 23, 25, 53, 80, 110, 135, 139, 143,
	443, 993, 995, 1433, 1521, 3000, 3306, 3389,
	5000, 5432, 5900, 6379, 8000, 8080, 8443, 9000,
	27017, // MongoDB
}

// ServiceMap maps port numbers to their common service names.
var ServiceMap = map[int]string{
	21:    "FTP",
	22:    "SSH",
	23:    "Telnet",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	110:   "POP3",
	135:   "RPC",
	139:   "NetBIOS",
	143:   "IMAP",
	443:   "HTTPS",
	993:   "IMAPS",
	995:   "POP3S",
	1433:  "MSSQL",
	1521:  "Oracle",
	3000:  "React/Node",
	3306:  "MySQL",
	3389:  "RDP",
	5000:  "Flask/Python",
	5432:  "PostgreSQL",
	5900:  "VNC",
	6379:  "Redis",
	8000:  "Django/Alt",
	8080:  "HTTP-Alt",
	8443:  "HTTPS-Alt",
	9000:  "PHP-FPM/Sonar",
	27017: "MongoDB",
}

// GetServiceName returns the common service name for a port, or "Unknown" if not found.
func GetServiceName(port int) string {
	if name, ok := ServiceMap[port]; ok {
		return name
	}
	return "Unknown"
}
