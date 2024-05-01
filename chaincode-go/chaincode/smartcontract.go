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
	Requests      []Request      `json:"requests"`
}

type HealthRecord struct {
	Description string `json:"description"`
	CreatedDate int64  `json:"createDate"`
	Date        int64  `json:"date"`
	EntityName  string `json:"entityName"`
	RecordType  string `json:"type"`
}

// Estrutura de Request
type Request struct {
	RequestID              string       `json:"requestID"`
	TypeOfAccess           TypeOfAccess `json:"typeOfAccess"`
	Description            string       `json:"description"`
	Organization           string       `json:"organization"`
	HealthCareProfessional string       `json:"healthcareProfessional"`
	SocialSecurityNumber   string       `json:"socialSecurityNumber"`
	CreatedDate            int64        `json:"createDate"`
	Status                 Status       `json:"status"`
	StatusChangedDate      int64        `json:"statusChangedDate"`
	ExpirationDate         int64        `json:"expirationDate"`
}

// Estrutura que define o estado do pedido
type Status int

const (
	Pending Status = iota
	Accepted
	Denied
)

type TypeOfAccess int

const (
	CreatePermission = 1 << iota // 1
	ReadPermission               // 2
	UpdatePermission             // 4
	DeletePermission             // 8
)

/*
	Vai existir um pedido efetuado pelo médico.
	O paciente aprova/recusa.

	Para isto vamos precisar:
	 - Método para ser possível o médico efetuar um pedido.  [X]
	 - Método para aprovar/rejeitar por parte do paciente. []
	 - Método para obter o pedido ao qual pretendemos aprovar/rejeitar. []
	 - Método para obter os dados médicos do paciente MAS apenas se existir uma autorização. []
*/

// Enviar um pedido ao cliente
func (c *HealthContract) RequestAccess(ctx contractapi.TransactionContextInterface,
	description, organization, socialSecurityNumber, healthCareProfessional string,
	expirationDate int64) error {

	// Cria uma instancia de Request e adiciona à lista de de ID's de pedidos efetuados
	// Como tem o time lá dentro, vai ser sempre único.
	request := Request{
		RequestID:              generateUniqueID(socialSecurityNumber),
		Description:            description,
		Organization:           organization,
		SocialSecurityNumber:   socialSecurityNumber,
		Status:                 Pending,
		HealthCareProfessional: healthCareProfessional,
		StatusChangedDate:      time.Now().Unix(),
		CreatedDate:            time.Now().Unix(),
		ExpirationDate:         expirationDate,
	}

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}
	patientWallet.Requests = append(patientWallet.Requests, request)

	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func (c *HealthContract) AddDataToWallet(ctx contractapi.TransactionContextInterface,
	content, healthCareProfessional, socialSecurityNumber, organization, recordType string, date int64) error {

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	err = checkIfHealthcareProfessionalHaveAccess(*patientWallet, organization, healthCareProfessional)

	if err != nil {
		return err
	}

	newRecord := HealthRecord{
		Description: content,
		CreatedDate: time.Now().Unix(),
		Date:        date,
		EntityName:  organization,
		RecordType:  recordType,
	}

	patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

// Vamos obter todo o histórico do utente.
func (c *HealthContract) GetMedicalHistory(ctx contractapi.TransactionContextInterface,
	organization, healthcareProfessional, socialSecurityNumber string) ([]HealthRecord, error) {

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	err = checkIfHealthcareProfessionalHaveAccess(*patientWallet, organization, healthcareProfessional)

	if err != nil {
		return nil, err
	}

	return patientWallet.HealthRecords, nil
}

// Função que permite o paciente aceitar ou negar o pedido de acesso aos seus dados
func (c *HealthContract) AnswerRequest(ctx contractapi.TransactionContextInterface,
	response Status, requestID, socialSecurityNumber string) error {

	// Colocar os erros de parametros 1.o não vale a pena ir à blockchain quando os campos nem válidos estão.
	if requestID == "" {
		return fmt.Errorf("invalid request ID: %s", requestID)
	}

	if socialSecurityNumber == "" {
		return fmt.Errorf("social security number cannot be empty")
	}

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	// Assim fica + elegante e provavelmente vamos precisar disto mais vezes.
	targetRequest := findPendingRequest(patientWallet, requestID)

	if targetRequest == nil {
		return fmt.Errorf("não existe nenhum pedido para atualizar, requestID: %s ", requestID)
	}

	if response == Denied {
		// Se o pedido for negado
		fmt.Println("Request denied for organization:", targetRequest.Organization)
	} else if response == Accepted {
		// Se o pedido for aceito
		fmt.Println("Request accepted for organization:", targetRequest.Organization)
	}

	// Volta a dar update ao estado do pedido de acesso
	targetRequest.Status = response
	targetRequest.StatusChangedDate = time.Now().Unix()

	// Vamos rezar para que ele seja inteligente e que esteja ligado no que diz respeito
	// a ser o mesmo objeto e desta forma poupo tempo na iteração etc.

	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func findPendingRequest(patientWallet *PatientWallet, requestID string) *Request {
	var targetRequest *Request

	for i := range patientWallet.Requests {
		_req := patientWallet.Requests[i]

		if _req.RequestID == requestID {
			targetRequest = &_req
			break
		}
	}

	return targetRequest
}

// func hasPermission(userPermission, requiredPermission int) bool {
// 	return userPermission&requiredPermission != 0
// }

func checkIfHealthcareProfessionalHaveAccess(patientWallet PatientWallet,
	organization, healthcareProfessional string) error {
	for _, request := range patientWallet.Requests {
		if request.Organization == organization &&
			request.HealthCareProfessional == healthcareProfessional &&
			request.ExpirationDate <= time.Now().Unix() {
			if request.Status == Accepted {
				return nil
			}
		}
	}
	return fmt.Errorf("no access found for organization: %s and healthcare professional: %s", organization, healthcareProfessional)
}

func createCompositeKey(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("PatientWallet", []string{"socialSecurityNumber", key})
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

func updateWallet(ctx contractapi.TransactionContextInterface, patientWallet PatientWallet, key string) error {
	updatedWalletBytes, err := json.Marshal(patientWallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated patient wallet: %v", err)
	}

	err = ctx.GetStub().PutState(key, updatedWalletBytes)
	if err != nil {
		return fmt.Errorf("failed to update patient wallet: %v", err)
	}

	return nil
}

func generateUniqueID(socialSecurityNumber string) string {
	// Add current timestamp to the input
	input := socialSecurityNumber + time.Now().String()

	// Hash the combined input
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashInBytes := hasher.Sum(nil)

	// Convert hash to hexadecimal string
	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}
