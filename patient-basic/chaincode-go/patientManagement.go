package main

import (
	"fmt"

        //"github.com/ApolloMedTech/FabricNetwork/chaincode-go/chaincode"
	"patientManagement.go/chaincode"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Método de start quando o chaincode leva deploy.
func main() {
	assetChaincode, err := contractapi.NewChaincode(&chaincode.HealthContract{})
	if err != nil {
		fmt.Printf("Error creating PatientChaincode: %v", err)
		return
	}

	if err := assetChaincode.Start(); err != nil {
		fmt.Printf("Error starting PatientChaincode: %v", err)
	}

	fmt.Printf("Se chegou aqui então correu bem e foi lançado corretamente")
}
