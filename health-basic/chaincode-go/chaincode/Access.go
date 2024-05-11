package chaincode

type Access struct {
	ResourceType             int    `json:"resourceType"` // 2
	RequestID                string `json:"requestID"`
	PatientID                string `json:"patientID"`
	PatientName              string `json:"patientName"`
	HealthcareProfessionalID string `json:"healthcareProfessionalID"`
	HealthcareProfessional   string `json:"healthcareProfessional"`
	CreatedDate              int64  `json:"createdDate"`
	ExpirationDate           int64  `json:"expirationDate"`
}
