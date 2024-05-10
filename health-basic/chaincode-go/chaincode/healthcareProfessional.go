package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type AddPatientMedicalRecordResponse struct {
	HealthcareProfessionalHasAccess bool `json:"healthcareProfessionalHasAccess"`
	HealthRecordAlreadyExist        bool `json:"healthRecordAlreadyExist"`
	HealthRecordAdded               bool `json:"healthRecordAdded"`
}

type GetPatientMedicalHistoryResponse struct {
	HealthcareProfessionalHasAccess bool           `json:"healthcareProfessionalHasAccess"`
	HealthRecords                   []HealthRecord `json:"healthRecords"`
}

type GetHealthRecordWithPHealthcareProfessionalByIDResponse struct {
	HealthcareProfessionalHasAccess bool         `json:"healthcareProfessionalHasAccess"`
	HealthRecord                    HealthRecord `json:"healthRecord"`
}

func (c *HealthContract) GetPatientMedicalHistory(ctx contractapi.TransactionContextInterface,
	patientID, healthcareProfessionalID string) (*GetPatientMedicalHistoryResponse, error) {

	resp := GetPatientMedicalHistoryResponse{}
	resp.HealthcareProfessionalHasAccess = checkIfHealthcareProfessionalHaveAccess(ctx, patientID, healthcareProfessionalID)

	if resp.HealthcareProfessionalHasAccess {
		healthRecords, err := c.GetMedicalHistory(ctx, patientID)
		if err != nil {
			return nil, fmt.Errorf("failed to get patient wallet: %v", err)
		}
		resp.HealthRecords = healthRecords
	}

	return &resp, nil
}

func (c *HealthContract) GetHealthRecordWithPHealthcareProfessionalByID(ctx contractapi.TransactionContextInterface, patientID, healthcareProfessionalID, recordID string) (*GetHealthRecordWithPHealthcareProfessionalByIDResponse, error) {

	resp := GetHealthRecordWithPHealthcareProfessionalByIDResponse{}
	resp.HealthcareProfessionalHasAccess = checkIfHealthcareProfessionalHaveAccess(ctx, patientID, healthcareProfessionalID)

	if resp.HealthcareProfessionalHasAccess {
		healthRecord, err := getHealthRecordByID(ctx, patientID, recordID)

		if err != nil {
			return nil, fmt.Errorf("erro ao obter o dado de saúde: %v", err)
		}

		resp.HealthRecord = *healthRecord
	}

	return &resp, nil
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

func (c *HealthContract) RequestPatientMedicalData(ctx contractapi.TransactionContextInterface,
	patientID, description, healthcareProfessionalID string,
	healthcareProfessional, requestID string) error {

	request := Request{
		ResourceType:             1,
		RequestID:                requestID,
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

func (c *HealthContract) GetRequestsWithHealthcareProfessional(ctx contractapi.TransactionContextInterface, healthcareProfessionalID string) ([]Request, error) {
	queryString := fmt.Sprintf(`{
        "selector": {
            "healthcareProfessionalID": "%s",
			"resourceType": 1,
			"status" : 0
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
	recordID, description, healthcareProfessionalID, healthcareProfessional, patientID,
	organization, recordType, speciality string, eventDate int64) (*AddPatientMedicalRecordResponse, error) {

	resp := AddPatientMedicalRecordResponse{}
	resp.HealthRecordAlreadyExist = checkIfHealthRecordAlreadyExist(ctx, recordID, patientID)
	resp.HealthcareProfessionalHasAccess = checkIfHealthcareProfessionalHaveAccess(ctx, patientID, healthcareProfessionalID)

	if !resp.HealthRecordAlreadyExist && resp.HealthcareProfessionalHasAccess {
		newRecord := HealthRecord{
			RecordID:                 recordID,
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
			return nil, fmt.Errorf("failed to create composite key: %v", err)
		}

		patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get patient wallet: %v", err)
		}

		patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

		if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
			return nil, fmt.Errorf("failed to update the wallet: %v", err)
		}

		resp.HealthRecordAdded = true
	}

	return &resp, nil
}

func checkIfHealthcareProfessionalHaveAccess(ctx contractapi.TransactionContextInterface, patientID, healthcareProfessionalID string) bool {
	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
            "healthcareProfessionalID": "%s",
            "resourceType": 2,
            "expirationDate": {
                "$gt": %d
            }
        }
    }`, patientID, healthcareProfessionalID, time.Now().Unix())

	return checkIfAnyDataAlreadyExist(ctx, queryString)
}
