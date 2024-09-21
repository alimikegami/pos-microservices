package paymentgateway

import (
	"github.com/alimikegami/point-of-sales/order-service/config"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
)

var midtransClient *coreapi.Client

func CreateMidtransClient(config *config.Config) *coreapi.Client {
	midtrans.ServerKey = config.MidtransConfig.ServerKey
	midtrans.Environment = midtrans.Sandbox // Use midtrans.Production for production

	midtransClient = &coreapi.Client{}
	midtransClient.New(midtrans.ServerKey, midtrans.Environment)

	return midtransClient
}
