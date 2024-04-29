package patient

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Patient struct {
	contractapi.Contract
}

type PatientWallet struct {
	HealthRecords []HealthRecord `json:"healthRecords"`
}

type HealthRecord struct {
	Description string `json:"description"`
	CreatedDate int64  `json:"createDate"`
	Date        int64  `json:"date"`
	EntityName  string `json:"entityName"`
	RecordType  string `json:"type"`
}

type AccessControls struct {
	AccessControls []AccessControl `json:"accessControls"`
}

type AccessControl struct {
	Description string `json:"description"`
	CreatedDate int64  `json:"createDate"`
	Date        int64  `json:"date"`
	EntityName  string `json:"entityName"`
	RecordType  string `json:"type"`
	RequestID   int    `json:"requestID"`
	Status      Status `json:"status"`
}

// HealthcareContract representa o smartcontract
type HealthcareContract struct {
	Requests []Request `json:"requests"` // Lista de ID's dos pedidos efetuados
}

// Estrutura de Request (não sei como deveria estar a organização)
type Request struct {
	Organization         string
	SocialSecurityNumber string
	Status               Status
}

// Estrutura que define o estado do pedido, começa por estar pendente, depois é aceite o negado.
type Status int

const (
	Pending Status = iota
	Accepted
	Denied
)

// Enviar um pedido ao cliente
func (c *Patient) SendRequest(ctx contractapi.TransactionContextInterface,
	organization, socialSecurityNumber string) error {
	// Cria uma instancia de Request e adiciona à lista de de ID's de pedidos efetuados
	request := Request{
		Organization:         organization,
		SocialSecurityNumber: socialSecurityNumber,
		Status:               Pending,
	}
	hc.Requests = append(hc.Requests, request)

	// Adicionar acessos à carteira do paciente quando o pedido é enviado
	err := hc.addAccessControl(ctx, "Content", socialSecurityNumber, "EntityName", "RecordType", time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to send an access request to %s: %v", socialSecurityNumber, err)
	}

	return nil
}

// Função que permite o paciente aceitar ou negar o pedido de acesso aos seus dados
func (c *Patient) RespondToRequest(ctx contractapi.TransactionContextInterface,
	requestID, response int, socialSecurityNumber string) error {
	// Encontrar o pedido com o ID correspondente
	var request *Request
	for i := range hc.Requests {
		if i == requestID {
			request = &hc.Requests[i]
			break
		}
	}

	if request == nil {
		return fmt.Errorf("request not found")
	}
	// Iterar sobre os acessos associados a este pedido
	accessControls, err := hc.getAccessControl(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to retrieve patient's access controls: %v", err)
	}

	// Verificar se a lista de acessos está vazia antes de iterar sobre ela
	if len(accessControls.AccessControls) == 0 {
		return fmt.Errorf("access controls list is empty")
	}

	// Atualizar o estado dos acessos associados ao pedido
	for i, accessControl := range accessControls.AccessControls {
		if accessControl.RequestID == requestID {
			accessControl.Status = Status(response)
			accessControls.AccessControls[i] = accessControl
			break
		}
	}

	// Atualizar os acessos
	if err := hc.updateAccessControl(ctx, *accessControls, socialSecurityNumber); err != nil {
		return fmt.Errorf("failed to update patient's access controls: %v", err)
	}

	if response == Denied {
		// Se o pedido for negado
		fmt.Println("Request denied for organization:", request.Organization)
	} else if response == Accepted {
		// Se o pedido for aceito
		fmt.Println("Request accepted for organization:", request.Organization)
	}

	// Volta a dar update ao estado do pedido de acesso
	request.Status = Status(response)

	return nil
}

// Adicionar um controle de acesso para o paciente
func (c *Patient) addAccessControl(ctx contractapi.TransactionContextInterface,
	content, socialSecurityNumber, entityName, recordType string, date int64) error {

	compositeKey, err := createCompositeKey(ctx, "AccessControl", socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	accessControl := AccessControl{
		Description: content,
		CreatedDate: time.Now().Unix(),
		Date:        date,
		EntityName:  entityName,
		RecordType:  recordType,
		RequestID:   len(hc.Requests) - 1, // Assume que o ID do pedido é o índice do último pedido adicionado
		Status:      Pending,
	}

	accessControls := AccessControls{
		AccessControls: []AccessControl{accessControl},
	}

	// Atualizar a lista de acessos associada à chave específica na blockchain
	if err := updateAccessControl(ctx, accessControls, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func (c *Patient) GetAccessControl(ctx contractapi.TransactionContextInterface,
	socialSecurityNumber string) ([]AccessControls, error) {

	compositeKey, err := createCompositeKey(ctx, "AccessControl", socialSecurityNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	accessControls := &AccessControls{
		AccessControls: patientWallet.AccessControls,
	}

	return accessControls, nil
}

// Vamos obter todo o histórico do utente.
func (c *Patient) GetMedicalHistory(ctx contractapi.TransactionContextInterface,
	socialSecurityNumber string) ([]HealthRecord, error) {

	compositeKey, err := createCompositeKey(ctx, "HealthRecord", socialSecurityNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	return patientWallet.HealthRecords, nil
}

// update de uma entrada da lista de acessos
// Atualiza a lista de acessos associada à chave especifíca na blockchain
func updateAccessControl(ctx contractapi.TransactionContextInterface,
	accessControls AccessControls, key string) error {
	accessControlsBytes, err := json.Marshal(accessControls)
	if err != nil {
		return fmt.Errorf("failed to marshal access controls: %v", err)
	}

	err = ctx.GetStub().PutState(key, accessControlsBytes)
	if err != nil {
		return fmt.Errorf("failed to update access controls: %v", err)
	}

	return nil
}

func (c *Patient) AddDataToWallet(ctx contractapi.TransactionContextInterface,
	content, socialSecurityNumber, entityName, recordType string, date int64) error {

	compositeKey, err := createCompositeKey(ctx, "HealthRecord", socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	newRecord := HealthRecord{
		Description: content,
		CreatedDate: time.Now().Unix(),
		Date:        date,
		EntityName:  entityName,
		RecordType:  recordType,
	}

	patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func createCompositeKey(ctx contractapi.TransactionContextInterface, schema, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey(schema, []string{"socialSecurityNumber", key})
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

func GenerateUniqueID(socialSecurityNumber string) string {
	hasher := sha256.New()
	hasher.Write([]byte(socialSecurityNumber)) // Write the data into the hasher
	hashInBytes := hasher.Sum(nil)             // Sum(nil) calculates the hash and returns bytes

	hashString := hex.EncodeToString(hashInBytes) // Convert bytes to hexadecimal string

	return hashString
}
