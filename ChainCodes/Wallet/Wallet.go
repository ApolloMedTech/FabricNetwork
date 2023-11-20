package wallet

import "time"

type PatientWallet struct {
	OwnerID       string                `json:"ownerID"`
	CreatedDate   time.Time             `json:"createdDate"`
	HealthRecords []HealthRecord        `json:"healthRecords"`
	Consents      []HealthRecordConsent `json:"consents"`
}

type HealthRecord struct {
	RecordID     string    `json:"recordID"`
	RecordTypeID int       `json:"recordType"`
	Content      string    `json:"content"`
	CreatedDate  time.Time `json:"createdDate"`
}

type HealthRecordConsent struct {
	OwnerID        string
	ConsentTypeID  int
	OrganizationID int
	userID         int
	CreatedDate    time.Time `json:"createdDate"`
	ExpirationDate time.Time `json:"expirationDate"`
}
