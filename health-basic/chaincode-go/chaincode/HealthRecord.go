package chaincode

type HealthRecord struct {
	RecordID                 string `json:"recordID"`
	Description              string `json:"description"`
	HealthCareProfessionalID string `json:"healthCareProfessionalID"`
	HealthCareProfessional   string `json:"healthCareProfessional"`
	CreatedDate              int64  `json:"createdDate"`
	EventDate                int64  `json:"eventDate"`
	Speciality               string `json:"speciality"`
	RecordType               string `json:"recordType"`
	Organization             string `json:"organization"`
}
