package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type User struct {
	Email     string `json:"email"`
	PatientID string `json:"patientID"`
}

type UserEnrollmentContract struct {
	contractapi.Contract
}

func (uec *UserEnrollmentContract) EnrollUser(ctx contractapi.TransactionContextInterface, email string, patientID string) error {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client ID: %s", err.Error())
	}

	existingUser, err := uec.UserExists(ctx, userID)
	if err != nil {
		return fmt.Errorf("error checking user existence: %s", err.Error())
	}
	if existingUser {
		return fmt.Errorf("user already enrolled")
	}

	user := User {
		Email:     email,
		PatientID: patientID
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user JSON: %s", err.Error())
	}

	err = ctx.GetStub().PutState(userID, userJSON)
	if err != nil {
		return fmt.Errorf("failed to put user on ledger: %s", err.Error())
	}

	return nil
}

func (uec *UserEnrollmentContract) UserExists(ctx contractapi.TransactionContextInterface, patientID string) (bool, error) {
	userJSON, err := ctx.GetStub().GetState(patientID)
	if err != nil {
		return false, fmt.Errorf("failed to read user from ledger: %s", err.Error())
	}
	return userJSON != nil, nil
}

func (uec *UserEnrollmentContract) GetUser(ctx contractapi.TransactionContextInterface, patientID string) (*User, error) {
	userJSON, err := ctx.GetStub().GetState(patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to read user from ledger: %s", err.Error())
	}
	if userJSON == nil {
		return nil, fmt.Errorf("user not found")
	}

	var user User
	err = json.Unmarshal(userJSON, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user JSON: %s", err.Error())
	}

	return &user, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&UserEnrollmentContract{})
	if err != nil {
		fmt.Printf("Error creating user enrollment chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting user enrollment chaincode: %s", err.Error())
	}
}
