package dataSources

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	plaid "github.com/plaid/plaid-go/v17/plaid"
)

var (
	PLAID_CLIENT_ID                      = ""
	PLAID_SECRET                         = ""
	PLAID_ENV                            = ""
	PLAID_PRODUCTS                       = "transactions"
	PLAID_COUNTRY_CODES                  = "US"
	PLAID_REDIRECT_URI                   = ""
	APP_PORT                             = ""
	client              *plaid.APIClient = nil
)

var environments = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"development": plaid.Development,
	"production":  plaid.Production,
}

// We store the access_token in memory - in production, store it in a secure
// persistent data store.
var accessToken string
var itemID string
var requestID string

var paymentID string

// The transfer_id is only relevant for the Transfer ACH product.
// We store the transfer_id in memory - in production, store it in a secure
// persistent data store
var transferID string

type access_token struct {
	Public_token string
}

func CreatePlaidClient() {
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", "64b714f4ef4079001a89cc37")
	configuration.AddDefaultHeader("PLAID-SECRET", "c56e69aa35b641d03481508a5502ef")
	configuration.UseEnvironment(plaid.Sandbox)
	client = plaid.NewAPIClient(configuration)
	fmt.Println("Plaid client created")
}

func CreateLinkToken(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	linkToken, err := linkTokenCreate(ctx, nil)
	if err != nil {
		fmt.Println(ctx, err, 68)
		return
	}
	fmt.Println(linkToken)

	w.WriteHeader(http.StatusOK)

	m := map[string]string{"link_token": linkToken}

	_ = json.NewEncoder(w).Encode(m)
}

// linkTokenCreate creates a link token using the specified parameters
func linkTokenCreate(
	ctx context.Context,
	paymentInitiation *plaid.LinkTokenCreateRequestPaymentInitiation,
) (string, error) {

	// Institutions from all listed countries will be shown.
	countryCodes := convertCountryCodes(strings.Split(PLAID_COUNTRY_CODES, ","))
	fmt.Println(countryCodes)
	redirectURI := PLAID_REDIRECT_URI

	// This should correspond to a unique id for the current user.
	// Typically, this will be a user ID number from your application.
	// Personally identifiable information, such as an email address or phone number, should not be used here.
	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: time.Now().String(),
	}

	request := plaid.NewLinkTokenCreateRequest(
		"Plaid Quickstart",
		"en",
		countryCodes,
		user,
	)

	if paymentInitiation != nil {
		request.SetPaymentInitiation(*paymentInitiation)
		// The 'payment_initiation' product has to be the only element in the 'products' list.
		request.SetProducts([]plaid.Products{plaid.PRODUCTS_PAYMENT_INITIATION})
	} else {
		products := convertProducts(strings.Split(PLAID_PRODUCTS, ","))
		fmt.Println(products)
		request.SetProducts(products)
	}

	if redirectURI != "" {
		request.SetRedirectUri(redirectURI)
	}

	linkTokenCreateResp, _, err := client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		fmt.Println(err, 122)
	}

	return linkTokenCreateResp.GetLinkToken(), nil
}

func CreatePublicToken(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Create a one-time use public_token for the Item.
	// This public_token can be used to initialize Link in update mode for a user
	publicTokenCreateResp, _, err := client.PlaidApi.ItemCreatePublicToken(ctx).ItemPublicTokenCreateRequest(
		*plaid.NewItemPublicTokenCreateRequest(accessToken),
	).Execute()

	if err != nil {
		fmt.Println(ctx, err, 139)
		return
	}

	fmt.Println(publicTokenCreateResp.GetPublicToken())
}

func GetAccessToken(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var t access_token
	body := json.NewDecoder(r.Body)
	err := body.Decode(&t)
	if err != nil {
		fmt.Println(err, 154)
		return
	}

	// exchange the public_token for an access_token
	exchangePublicTokenResp, _, err := client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(t.Public_token),
	).Execute()
	if err != nil {
		fmt.Println(err, 163)
		return
	}

	accessToken = exchangePublicTokenResp.GetAccessToken()
	itemID = exchangePublicTokenResp.GetItemId()
	requestID = exchangePublicTokenResp.GetRequestId()
	if itemExists(strings.Split(PLAID_PRODUCTS, ","), "transfer") {
		transferID, err = authorizeAndCreateTransfer(ctx, client, accessToken)
	}

	fmt.Println()
	fmt.Println("public token: " + t.Public_token)
	fmt.Println("access token: " + accessToken)
	fmt.Println("item ID: " + itemID)
	fmt.Println("request ID: " + requestID)
	fmt.Println()

	m := map[string]string{"access_token": accessToken, "itemID": itemID, "requestID": requestID}

	_ = json.NewEncoder(w).Encode(m)
}

func Auth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	authGetResp, _, err := client.PlaidApi.AuthGet(ctx).AuthGetRequest(
		*plaid.NewAuthGetRequest(accessToken),
	).Execute()

	if err != nil {
		fmt.Println(ctx, err, 195)
		return
	}

	fmt.Println("accounts", authGetResp.GetAccounts())
	fmt.Println("numbers", authGetResp.GetNumbers())
}

func Accounts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	accountsGetResp, _, err := client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	if err != nil {
		fmt.Println(err, 212)
		w.Header().Set("plaid error", strings.Split(err.Error(), " ")[0])
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Println("accounts", accountsGetResp.GetAccounts())
	_ = json.NewEncoder(w).Encode(accountsGetResp.GetAccounts())
}

func Balance(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	balancesGetResp, _, err := client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(
		*plaid.NewAccountsBalanceGetRequest(accessToken),
	).Execute()

	if err != nil {
		fmt.Println(ctx, err, 228)
		return
	}

	fmt.Println("accounts", balancesGetResp)
	_ = json.NewEncoder(w).Encode(balancesGetResp.GetAccounts())
}

func Transactions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Set cursor to empty to receive all historical updates
	var cursor *string

	// New transaction updates since "cursor"
	var added []plaid.Transaction
	var modified []plaid.Transaction
	var removed []plaid.RemovedTransaction // Removed transaction ids
	hasMore := true
	// Iterate through each page of new transaction updates for item
	for hasMore {
		request := plaid.NewTransactionsSyncRequest(accessToken)
		if cursor != nil {
			request.SetCursor(*cursor)
		}
		resp, _, err := client.PlaidApi.TransactionsSync(
			ctx,
		).TransactionsSyncRequest(*request).Execute()
		if err != nil {
			fmt.Println(ctx, err, 258)
			return
		}

		// Add this page of results
		added = append(added, resp.GetAdded()...)
		modified = append(modified, resp.GetModified()...)
		removed = append(removed, resp.GetRemoved()...)
		hasMore = resp.GetHasMore()
		// Update cursor to the next cursor
		nextCursor := resp.GetNextCursor()
		cursor = &nextCursor
	}

	sort.Slice(added, func(i, j int) bool {
		return added[i].GetDate() < added[j].GetDate()
	})
	latestTransactions := added[len(added)-9:]

	fmt.Println("latest_transactions", latestTransactions)
	_ = json.NewEncoder(w).Encode(latestTransactions)
}

func InvestmentTransactions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	endDate := time.Now().Local().Format("2006-01-02")
	startDate := time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02")

	request := plaid.NewInvestmentsTransactionsGetRequest(accessToken, startDate, endDate)
	invTxResp, _, err := client.PlaidApi.InvestmentsTransactionsGet(ctx).InvestmentsTransactionsGetRequest(*request).Execute()

	if err != nil {
		fmt.Println(ctx, err, 292)
		return
	}

	fmt.Println(invTxResp)
}

func Holdings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	holdingsGetResp, _, err := client.PlaidApi.InvestmentsHoldingsGet(ctx).InvestmentsHoldingsGetRequest(
		*plaid.NewInvestmentsHoldingsGetRequest(accessToken),
	).Execute()
	if err != nil {
		fmt.Println(err, 307)
		return
	}

	fmt.Println(holdingsGetResp)
	_ = json.NewEncoder(w).Encode(holdingsGetResp)
}

func Info(w http.ResponseWriter, r *http.Request) {
	item := map[string]interface{}{
		"item_id":      itemID,
		"access_token": accessToken,
		"products":     strings.Split(PLAID_PRODUCTS, ","),
	}

	fmt.Println(item)

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(item)
}

func convertCountryCodes(countryCodeStrs []string) []plaid.CountryCode {
	countryCodes := []plaid.CountryCode{}

	for _, countryCodeStr := range countryCodeStrs {
		countryCodes = append(countryCodes, plaid.CountryCode(countryCodeStr))
	}

	return countryCodes
}

func convertProducts(productStrs []string) []plaid.Products {
	products := []plaid.Products{}

	for _, productStr := range productStrs {
		products = append(products, plaid.Products(productStr))
	}

	return products
}

func Assets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	createRequest := plaid.NewAssetReportCreateRequest(10)
	createRequest.SetAccessTokens([]string{accessToken})

	// create the asset report
	assetReportCreateResp, _, err := client.PlaidApi.AssetReportCreate(ctx).AssetReportCreateRequest(
		*createRequest,
	).Execute()
	if err != nil {
		fmt.Println(ctx, err, 361)
		return
	}

	assetReportToken := assetReportCreateResp.GetAssetReportToken()

	// get the asset report
	assetReportGetResp, err := pollForAssetReport(ctx, client, assetReportToken)
	if err != nil {
		fmt.Println(ctx, err)
		return
	}

	// get it as a pdf
	pdfRequest := plaid.NewAssetReportPDFGetRequest(assetReportToken)
	pdfFile, _, err := client.PlaidApi.AssetReportPdfGet(ctx).AssetReportPDFGetRequest(*pdfRequest).Execute()
	if err != nil {
		fmt.Println(ctx, err, 378)
		return
	}

	reader := bufio.NewReader(pdfFile)
	content, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println(ctx, err)
		return
	}

	// convert pdf to base64
	encodedPdf := base64.StdEncoding.EncodeToString(content)

	fmt.Println(assetReportGetResp.GetReport(), encodedPdf)
	_ = json.NewEncoder(w).Encode(encodedPdf)
}

func pollForAssetReport(ctx context.Context, client *plaid.APIClient, assetReportToken string) (*plaid.AssetReportGetResponse, error) {
	numRetries := 20
	request := plaid.NewAssetReportGetRequest()
	request.SetAssetReportToken(assetReportToken)

	for i := 0; i < numRetries; i++ {
		response, _, err := client.PlaidApi.AssetReportGet(ctx).AssetReportGetRequest(*request).Execute()
		if err != nil {
			plaidErr, err := plaid.ToPlaidError(err)
			if plaidErr.ErrorCode == "PRODUCT_NOT_READY" {
				time.Sleep(1 * time.Second)
				continue
			} else {
				return nil, err
			}
		} else {
			return &response, nil
		}
	}
	return nil, errors.New("Timed out when polling for an asset report.")
}

// This is a helper function to authorize and create a Transfer after successful
// exchange of a public_token for an access_token. The transfer_id is then used
// to obtain the data about that particular Transfer.
func authorizeAndCreateTransfer(ctx context.Context, client *plaid.APIClient, accessToken string) (string, error) {
	// We call /accounts/get to obtain first account_id - in production,
	// account_id's should be persisted in a data store and retrieved
	// from there.
	accountsGetResp, _, _ := client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	accountID := accountsGetResp.GetAccounts()[0].AccountId
	transferType, err := plaid.NewTransferTypeFromValue("debit")
	transferNetwork, err := plaid.NewTransferNetworkFromValue("ach")
	ACHClass, err := plaid.NewACHClassFromValue("ppd")

	transferAuthorizationCreateUser := plaid.NewTransferAuthorizationUserInRequest("FirstName LastName")
	transferAuthorizationCreateRequest := plaid.NewTransferAuthorizationCreateRequest(
		accessToken,
		accountID,
		*transferType,
		*transferNetwork,
		".01",
		*transferAuthorizationCreateUser)

	transferAuthorizationCreateRequest.SetAchClass(*ACHClass)

	transferAuthorizationCreateResp, _, err := client.PlaidApi.TransferAuthorizationCreate(ctx).TransferAuthorizationCreateRequest(*transferAuthorizationCreateRequest).Execute()
	if err != nil {
		fmt.Println(err, 447)
	}
	authorizationID := transferAuthorizationCreateResp.GetAuthorization().Id

	transferCreateRequest := plaid.NewTransferCreateRequest(
		authorizationID,
		"Debit",
	)

	transferCreateRequest.SetAccessToken(accessToken)
	transferCreateRequest.SetAccountId(accountID)

	transferCreateResp, _, err := client.PlaidApi.TransferCreate(ctx).TransferCreateRequest(*transferCreateRequest).Execute()
	if err != nil {
		fmt.Println(err, 461)
	}

	return transferCreateResp.GetTransfer().Id, nil
}

// Helper function to determine if Transfer is in Plaid product array
func itemExists(array []string, product string) bool {
	for _, item := range array {
		if item == product {
			return true
		}
	}

	return false
}
