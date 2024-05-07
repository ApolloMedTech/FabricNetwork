package chaincode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type HealthContract struct {
	contractapi.Contract
}

type PatientWallet struct {
	HealthRecords []HealthRecord `json:"healthRecords"`
}

type HealthRecord struct {
	Description              string `json:"description"`
	HealthCareProfessionalID string `json:"healthCareProfessionalID"`
	HealthCareProfessional   string `json:"healthCareProfessional"`
	CreatedDate              int64  `json:"createdDate"`
	EventDate                int64  `json:"eventDate"`
	Speciality               string `json:"speciality"`
	RecordType               string `json:"recordType"`
	Organization             string `json:"organization"`
}

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
}

type Access struct {
	ResourceType             int    `json:"resourceType"` // 2
	RequestID                string `json:"requestID"`
	PatientID                string `json:"patientID"`
	HealthcareProfessionalID string `json:"healthcareProfessionalID"`
	CreatedDate              int64  `json:"createdDate"`
	ExpirationDate           int64  `json:"expirationDate"`
}

func (c *HealthContract) GetPatientMedicalHistory(ctx contractapi.TransactionContextInterface,
	patientID, healthcareProfessionalID string) ([]HealthRecord, error) {

	err := checkIfHealthcareProfessionalHaveAccess(ctx, patientID, healthcareProfessionalID)

	if err != nil {
		return nil, err
	}

	healthRecords, err := c.GetMedicalHistory(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient wallet: %v", err)
	}

	return healthRecords, nil
}

func (c *HealthContract) GetMedicalHistory(ctx contractapi.TransactionContextInterface, patientID string) ([]HealthRecord, error) {

	// Injeto o ID da wallet e assim é mais rápido.
	queryString := fmt.Sprintf(`{
        "selector": {
            "_id": "\u0000%s\u0000%s\u0000%s\u0000"
        },
		"fields": [
			"healthRecords"
		]
    }`, "PatientWallet", "patientID", patientID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)

	// Aqui não posso dar erro, tenho de fazer desta maneira
	if err != nil {
		return []HealthRecord{}, nil
	}
	defer queryResultsIterator.Close()

	var patientWallet PatientWallet
	var healthRecords []HealthRecord

	if queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()

		if err != nil {
			return nil, fmt.Errorf("erro ao obter os dados do paciente: %v", err)
		}

		if err := json.Unmarshal(queryResponse.Value, &patientWallet); err != nil {
			return nil, fmt.Errorf("erro ao transformar os dados na wallet: %v", err)
		}

		healthRecords = patientWallet.HealthRecords
	} else {
		healthRecords = []HealthRecord{}
	}

	return healthRecords, nil
}

// getAccessesByPatientID retrieves all accesses associated with a specific patient ID using a selector query.
func (c *HealthContract) GetAccessesByPatientID(ctx contractapi.TransactionContextInterface, patientID string) ([]Access, error) {
	// Construct the selector query to retrieve accesses by patientID
	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
			"resourceType": 2
        }
    }`, patientID)

	// Execute the selector query
	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return []Access{}, nil
	}
	defer queryResultsIterator.Close()

	// Initialize an empty slice to store retrieved accesses
	var accesses []Access

	// Iterate through the query results
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error retrieving next query result: %v", err)
		}

		// Deserialize the access from JSON
		var access Access
		err = json.Unmarshal(queryResponse.Value, &access)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling access: %v", err)
		}

		// Append the retrieved access to the slice
		accesses = append(accesses, access)
	}

	// Return the retrieved accesses
	return accesses, nil
}

func (c *HealthContract) GetAccessesByHealthcareProfessionalID(ctx contractapi.TransactionContextInterface,
	healthcareProfessionalID string) ([]Access, error) {
	// Construct the selector query to retrieve accesses by patientID
	queryString := fmt.Sprintf(`{
        "selector": {
            "healthcareProfessionalID": "%s",
			"resourceType": 2
        }
    }`, healthcareProfessionalID)

	// Execute the selector query
	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return []Access{}, nil
	}
	defer queryResultsIterator.Close()

	// Initialize an empty slice to store retrieved accesses
	var accesses []Access

	// Iterate through the query results
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error retrieving next query result: %v", err)
		}

		// Deserialize the access from JSON
		var access Access
		err = json.Unmarshal(queryResponse.Value, &access)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling access: %v", err)
		}

		// Append the retrieved access to the slice
		accesses = append(accesses, access)
	}

	// Return the retrieved accesses
	return accesses, nil
}

// AnswerRequest allows the patient to accept or deny the request for access to their data.
func (c *HealthContract) AnswerRequest(ctx contractapi.TransactionContextInterface,
	response int, requestID, patientID string, expirationDate int64) error {

	// Check parameter validity
	if requestID == "" {
		return fmt.Errorf("invalid request ID: %s", requestID)
	}

	if patientID == "" {
		return fmt.Errorf("social security number cannot be empty")
	}

	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
			"requestID": "%s",
			"resourceType": 1
        }
    }`, patientID, requestID)

	// Execute the query
	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}

	defer queryResultsIterator.Close()

	if queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return fmt.Errorf("error retrieving next query result: %v", err)
		}

		var request Request
		err = json.Unmarshal(queryResponse.Value, &request)
		if err != nil {
			return fmt.Errorf("error unmarshalling query result: %v", err)
		}

		request.Status = response
		request.StatusChangedDate = time.Now().Unix()

		updatedRequestJSON, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to marshal updated request: %v", err)
		}

		// Update the request on the ledger
		err = ctx.GetStub().PutState(queryResponse.Key, updatedRequestJSON)
		if err != nil {
			return fmt.Errorf("failed to update request: %v", err)
		}

		if response == 1 {
			err := addAccess(ctx, requestID, patientID, request.HealthcareProfessionalID, expirationDate)
			if err != nil {
				return fmt.Errorf("failed to add access: %v", err)
			}
		}
	}

	return nil
}

// addAccess adds access to the patient's data.
func addAccess(ctx contractapi.TransactionContextInterface, requestID, patientID, healthcareProfessionalID string, expirationDate int64) error {
	// Create a new access based on the approved request
	access := Access{
		ResourceType:             2,
		RequestID:                requestID,
		PatientID:                patientID,
		HealthcareProfessionalID: healthcareProfessionalID,
		CreatedDate:              time.Now().Unix(),
		ExpirationDate:           expirationDate, // or set the expiration date as needed
	}

	// Serialize the access object to JSON
	accessJSON, err := json.Marshal(access)
	if err != nil {
		return fmt.Errorf("failed to serialize access to JSON: %v", err)
	}

	// Generate composite key for the access
	compositeKey, err := createAcessesCompositeKey(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to create composite key for access: %v", err)
	}

	// Store the serialized access on the ledger
	err = ctx.GetStub().PutState(compositeKey, accessJSON)
	if err != nil {
		return fmt.Errorf("failed to store access on the ledger: %v", err)
	}

	return nil
}

func (c *HealthContract) RequestPatientMedicalData(ctx contractapi.TransactionContextInterface,
	patientID, description, healthcareProfessionalID string,
	healthcareProfessional string) error {

	request := Request{
		ResourceType:             1,
		RequestID:                generateUniqueID(patientID, description),
		Description:              description,
		PatientID:                patientID,
		Status:                   0,
		HealthcareProfessionalID: healthcareProfessionalID,
		HealthcareProfessional:   healthcareProfessional,
		StatusChangedDate:        time.Now().Unix(),
		CreatedDate:              time.Now().Unix(),
	}

	err := storeRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to store request: %v", err)
	}

	return nil
}

func (c *HealthContract) GetRequestsWithPatient(ctx contractapi.TransactionContextInterface, patientID string) ([]Request, error) {
	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
			"resourceType": 1
        }
    }`, patientID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return []Request{}, nil
	}
	defer queryResultsIterator.Close()

	var requests []Request
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error retrieving next query result: %v", err)
		}

		var request Request
		if err := json.Unmarshal(queryResponse.Value, &request); err != nil {
			return nil, fmt.Errorf("error unmarshalling query result: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func (c *HealthContract) GetRequestsWithHealthcareProfessional(ctx contractapi.TransactionContextInterface, healthcareProfessionalID string) ([]Request, error) {
	queryString := fmt.Sprintf(`{
        "selector": {
            "healthcareProfessionalID": "%s",
			"resourceType": 1
        }
    }`, healthcareProfessionalID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return []Request{}, nil
	}
	defer queryResultsIterator.Close()

	var requests []Request
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error retrieving next query result: %v", err)
		}

		var request Request
		if err := json.Unmarshal(queryResponse.Value, &request); err != nil {
			return nil, fmt.Errorf("error unmarshalling query result: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func (c *HealthContract) AddPatientMedicalRecord(ctx contractapi.TransactionContextInterface,
	description, healthcareProfessionalID, healthcareProfessional, patientID,
	organization, recordType, speciality string, eventDate int64) error {

	err := checkIfHealthcareProfessionalHaveAccess(ctx, patientID, healthcareProfessionalID)

	if err != nil {
		return err
	}

	newRecord := HealthRecord{
		Description:              description,
		CreatedDate:              time.Now().Unix(),
		HealthCareProfessional:   healthcareProfessional,
		HealthCareProfessionalID: healthcareProfessionalID,
		EventDate:                eventDate,
		Organization:             organization,
		RecordType:               recordType,
		Speciality:               speciality,
	}

	compositeKey, err := createPatientWalletCompositeKey(ctx, patientID)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to get patient wallet: %v", err)
	}

	patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func updateWallet(ctx contractapi.TransactionContextInterface, patientWallet PatientWallet, patientID string) error {
	updatedWallet, err := json.Marshal(patientWallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated health records: %v", err)
	}

	err = ctx.GetStub().PutState(patientID, updatedWallet)
	if err != nil {
		return fmt.Errorf("failed to update health records: %v", err)
	}

	return nil
}

func storeRequest(ctx contractapi.TransactionContextInterface, request Request) error {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to serialize request to JSON: %v", err)
	}

	compositeKey, err := createRequestCompositeKey(ctx, request.PatientID, request.HealthcareProfessionalID)

	if err != nil {
		return fmt.Errorf("failed to create composite key for request: %v", err)
	}

	err = ctx.GetStub().PutState(compositeKey, requestJSON)
	if err != nil {
		return fmt.Errorf("failed to store request on the ledger: %v", err)
	}

	return nil
}

func createPatientWalletCompositeKey(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("PatientWallet", []string{"patientID", key})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func createRequestCompositeKey(ctx contractapi.TransactionContextInterface, patientID, healthcareProfessionalID string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Requests", []string{"patientID", patientID, "healthcareProfessionalID", healthcareProfessionalID})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func createAcessesCompositeKey(ctx contractapi.TransactionContextInterface, requestID string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Acesses", []string{"requestID", requestID})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func getCurrentPatientWallet(ctx contractapi.TransactionContextInterface,
	key string) (*PatientWallet, error) {

	walletBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read patient wallet: %v", err)
	}

	var patientWallet PatientWallet

	if walletBytes != nil {
		err = json.Unmarshal(walletBytes, &patientWallet)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
		}
	} else {
		patientWallet.HealthRecords = []HealthRecord{}
	}

	return &patientWallet, nil
}

func checkIfHealthcareProfessionalHaveAccess(ctx contractapi.TransactionContextInterface, patientID, healthcareProfessionalID string) error {

	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
            "healthcareProfessionalID": "%s",
			"resourceType": 2
        }
    }`, patientID, healthcareProfessionalID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer queryResultsIterator.Close()

	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return fmt.Errorf("error retrieving next query result: %v", err)
		}

		var access Access
		if err := json.Unmarshal(queryResponse.Value, &access); err != nil {
			return fmt.Errorf("error unmarshalling query result: %v", err)
		}

		// inverter o if quando no servidor :)
		if access.ExpirationDate <= time.Now().Unix() {
			return nil
		}
	}

	return fmt.Errorf("have no access %v", err)
}

func generateUniqueID(patientID, description string) string {
	input := patientID + description

	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashInBytes := hasher.Sum(nil)

	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}
