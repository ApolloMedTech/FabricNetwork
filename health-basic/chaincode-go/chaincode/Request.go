package chaincode

type Request struct {
	ResourceType             int    `json:"resourceType"` // 1
	RequestID                string `json:"requestID"`
	Description              string `json:"description"`
	HealthcareProfessionalID string `json:"healthcareProfessionalID"`
	HealthcareProfessional   string `json:"healthcareProfessional"`
	PatientID                string `json:"patientID"`
	CreatedDate              int64  `json:"createdDate"`
	Status                   int    `json:"status"`
	StatusChangedDate        int64  `json:"statusChangedDate"`
	ExpirationDate           int64  `json:"expirationDate"`
}
