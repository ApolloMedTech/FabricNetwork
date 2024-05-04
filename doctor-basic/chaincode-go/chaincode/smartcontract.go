package chaincode

// Doctor smart-contract
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type DoctorContract struct {
	contractapi.Contract
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

// Estrutura de Request
type Request struct {
	RequestID                string `json:"requestID"`
	Description              string `json:"description"`
	Organization             string `json:"organization"`
	HealthCareProfessionalID string `json:"healthcareProfessionalID"`
	HealthCareProfessional   string `json:"healthcareProfessional"`
	PatientID                string `json:"patientID"`
	CreatedDate              int64  `json:"createdDate"`
	Status                   int    `json:"status"`
	StatusChangedDate        int64  `json:"statusChangedDate"`
}

type Access struct {
	RequestID                string `json:"requestID"`
	PatientID                string `json:"patientID"`
	HealthcareProfessionalID string `json:"healthcareProfessionalID"`
	Organization             string `json:"organization"`
	CreatedDate              int64  `json:"createdDate"`
	ExpirationDate           int64  `json:"expirationDate"`
}

// Vamos obter todo o histórico do utente.
func (c *DoctorContract) GetPatientMedicalHistory(ctx contractapi.TransactionContextInterface,
	patientID, healthcareProfessionalID, organization string) ([]HealthRecord, error) {

	err := checkAuthorizationVerification(ctx,
		organization,
		healthcareProfessionalID,
		patientID)

	if err != nil {
		return nil, err
	}

	healthRecordsKey := fmt.Sprintf("healthRecords_%s", patientID) // Assuming patientID is unique
	existingRecordsBytes, err := ctx.GetStub().GetState(healthRecordsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing health records: %v", err)
	}

	var existingRecords []HealthRecord
	if existingRecordsBytes != nil {
		if err := json.Unmarshal(existingRecordsBytes, &existingRecords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal existing health records: %v", err)
		}
	}

	return existingRecords, nil
}

// Enviar um pedido ao cliente
func (c *DoctorContract) RequestPatientMedicalData(ctx contractapi.TransactionContextInterface,
	patientID, description, organization, healthCareProfessionalID string,
	healthCareProfessional string) error {

	// Cria uma instancia de Request e adiciona à lista de de ID's de pedidos efetuados
	// Como tem o time lá dentro, vai ser sempre único.
	request := Request{
		RequestID:                generateUniqueID(patientID, description),
		Description:              description,
		Organization:             organization,
		PatientID:                patientID,
		Status:                   0,
		HealthCareProfessionalID: healthCareProfessionalID,
		HealthCareProfessional:   healthCareProfessional,
		StatusChangedDate:        time.Now().Unix(),
		CreatedDate:              time.Now().Unix(),
	}

	err := storeRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to store request: %v", err)
	}

	return nil
}

func storeRequest(ctx contractapi.TransactionContextInterface, request Request) error {
	// Serialize the request object to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to serialize request to JSON: %v", err)
	}

	// Generate composite key for the request
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Request", []string{request.RequestID})
	if err != nil {
		return fmt.Errorf("failed to create composite key for request: %v", err)
	}

	// Store the serialized request on the ledger
	err = ctx.GetStub().PutState(compositeKey, requestJSON)
	if err != nil {
		return fmt.Errorf("failed to store request on the ledger: %v", err)
	}

	return nil
}

func (c *DoctorContract) AddPatientMedicalRecord(ctx contractapi.TransactionContextInterface,
	description, healthCareProfessionalID, healthCareProfessional, patientID,
	organization, recordType, speciality string, eventDate int64) error {

	compositeKey, err := createCompositeKey(ctx, patientID)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	acesses, err := getCurrentPatientAccesses(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	err = checkIfHealthcareProfessionalHaveAccess(*acesses, organization, healthCareProfessionalID)

	if err != nil {
		return err
	}

	newRecord := HealthRecord{
		Description:              description,
		CreatedDate:              time.Now().Unix(),
		HealthCareProfessional:   healthCareProfessional,
		HealthCareProfessionalID: healthCareProfessionalID,
		EventDate:                eventDate,
		Organization:             organization,
		RecordType:               recordType,
		Speciality:               speciality,
	}

	healthRecordsKey := fmt.Sprintf("healthRecords_%s", patientID) // Assuming patientID is unique
	existingRecordsBytes, err := ctx.GetStub().GetState(healthRecordsKey)
	if err != nil {
		return fmt.Errorf("failed to read existing health records: %v", err)
	}

	var existingRecords []HealthRecord
	if existingRecordsBytes != nil {
		if err := json.Unmarshal(existingRecordsBytes, &existingRecords); err != nil {
			return fmt.Errorf("failed to unmarshal existing health records: %v", err)
		}
	}

	existingRecords = append(existingRecords, newRecord)

	if err := updateHealthRecords(ctx, existingRecords, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

// Vamos obter todos os pedidos efetuados ao paciente.
func (c *DoctorContract) GetRequests(ctx contractapi.TransactionContextInterface, healthCareProfessionalID, organization string) ([]Request, error) {
	queryString := fmt.Sprintf(`{
        "selector": {
            "healthCareProfessionalID": "%s",
            "organization": "%s"
        }
    }`, healthCareProfessionalID, organization)

	// Execute the query
	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}

	defer queryResultsIterator.Close()

	// Iterate over the query results and parse them into Request structs
	var requests []Request
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error retrieving next query result: %v", err)
		}

		// Unmarshal the query response into a Request struct
		var request Request
		err = json.Unmarshal(queryResponse.Value, &request)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling query result: %v", err)
		}

		// Append the request to the list of requests
		requests = append(requests, request)
	}

	return requests, nil
}

func checkAuthorizationVerification(ctx contractapi.TransactionContextInterface,
	organization, healthcareProfessionalID, patientID string) error {

	compositeKey, err := createCompositeKey(ctx, patientID)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	acesses, err := getCurrentPatientAccesses(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	err = checkIfHealthcareProfessionalHaveAccess(*acesses, organization, healthcareProfessionalID)

	if err != nil {
		return err
	}

	return nil
}

func checkIfHealthcareProfessionalHaveAccess(accesses []Access, organization, healthcareProfessionalID string) error {
	for _, access := range accesses {
		if access.Organization == organization &&
			access.HealthcareProfessionalID == healthcareProfessionalID &&
			access.ExpirationDate <= time.Now().Unix() { // Inverter o sinal quando tivermos em PRD.
			return nil
		}
	}
	return fmt.Errorf("no access found for organization: %s and healthcare professional: %s", organization, healthcareProfessionalID)
}

func createCompositeKey(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("HealthRecord", []string{"patientID", key})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func getCurrentPatientAccesses(ctx contractapi.TransactionContextInterface, key string) (*[]Access, error) {

	walletBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read patient wallet: %v", err)
	}

	var acesses []Access

	if walletBytes != nil {
		err = json.Unmarshal(walletBytes, &acesses)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
		}
	} else {
		acesses = []Access{}
	}

	return &acesses, nil
}

func updateHealthRecords(ctx contractapi.TransactionContextInterface, healthRecords []HealthRecord, patientID string) error {
	updatedRecordsBytes, err := json.Marshal(healthRecords)
	if err != nil {
		return fmt.Errorf("failed to marshal updated health records: %v", err)
	}

	err = ctx.GetStub().PutState(patientID, updatedRecordsBytes)
	if err != nil {
		return fmt.Errorf("failed to update health records: %v", err)
	}

	return nil
}

func generateUniqueID(patientID, description string) string {
	input := patientID + description

	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashInBytes := hasher.Sum(nil)

	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}
