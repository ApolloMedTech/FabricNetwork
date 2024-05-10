package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

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

func (c *HealthContract) GetHealthRecordWithPatientByID(ctx contractapi.TransactionContextInterface, patientID, recordID string) (*HealthRecord, error) {

	healthRecord, err := getHealthRecordByID(ctx, patientID, recordID)

	if err != nil {
		return nil, fmt.Errorf("erro ao obter o dado de saúde: %v", err)
	}

	return healthRecord, nil
}

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

func (c *HealthContract) RemoveAccess(ctx contractapi.TransactionContextInterface, patientID, requestID string) error {

	queryString := fmt.Sprintf(`{
        "selector": {
            "_id": "\u0000%s\u0000%s\u0000%s\u0000",
			"patientID": "%s",
			"resourceType": 2
        }
    }`, "Acesses", "requestID", requestID, patientID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("no access found: %v", err)
	}

	if queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return fmt.Errorf("error retrieving next query result: %v", err)
		}

		var access Access
		err = json.Unmarshal(queryResponse.Value, &access)
		if err != nil {
			return fmt.Errorf("error unmarshalling access: %v", err)
		}

		// Expiramos o acesso, simplificado por agora.
		access.ExpirationDate = time.Now().Unix()

		updatedAccessJSON, err := json.Marshal(access)
		if err != nil {
			return fmt.Errorf("failed to marshal updated request: %v", err)
		}

		// Update the request on the ledger
		err = ctx.GetStub().PutState(queryResponse.Key, updatedAccessJSON)
		if err != nil {
			return fmt.Errorf("failed to update request: %v", err)
		}
	}

	return nil
}

func (c *HealthContract) GetRequestsWithPatient(ctx contractapi.TransactionContextInterface, patientID string) ([]Request, error) {
	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
			"resourceType": 1,
			"status" : 0
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
			err := addAccess(ctx, requestID, patientID, request.HealthcareProfessionalID, request.HealthcareProfessional, expirationDate)
			if err != nil {
				return fmt.Errorf("failed to add access: %v", err)
			}
		}
	}

	return nil
}
