package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/escalopa/gobank/db/mock"
	db "github.com/escalopa/gobank/db/sqlc"
	"github.com/escalopa/gobank/token"
	"github.com/escalopa/gobank/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	user, _ := createRandomUser(t)
	account := createRandomAccount(user.Username)
	accRes := mapAccountToResponse(account)

	arg := createAccountReq{
		Currency: account.Currency,
	}

	testCases := []struct {
		name       string
		accountArg createAccountReq
		testCaseBase
	}{
		{
			name:       "OK",
			accountArg: arg,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
							Owner:    user.Username,
							Balance:  0,
							Currency: arg.Currency,
						})).
						Times(1).
						Return(account, nil)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusCreated, recorder.Code)
					requireBodyMatchAccount(t, recorder.Body, accRes)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name:       "BadRequest",
			accountArg: createAccountReq{},
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						CreateAccount(gomock.Any(), gomock.Any()).
						Times(0)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name:       "InternalError",
			accountArg: arg,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						CreateAccount(gomock.Any(), gomock.Any()).
						Times(1).
						Return(db.Account{}, sql.ErrConnDone)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusInternalServerError, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
	}

	for i := 0; i < len(testCases); i++ {
		tc := testCases[i]

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(tc.accountArg)
		require.NoError(t, err)

		url := "/api/accounts"
		reader := bytes.NewReader(buf.Bytes())

		req, err := http.NewRequest(http.MethodPost, url, reader)
		require.NoError(t, err)

		runServerTest(t, tc, req)
	}
}

func createRandomAccount(owner string) db.Account {
	return db.Account{
		ID:       util.RandomInteger(1, 1000),
		Owner:    owner,
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
}

func requireBodyMatchAccount(t *testing.T, b *bytes.Buffer, account *accountResponse) {
	data, err := io.ReadAll(b)
	require.NoError(t, err)

	var accountReceived accountResponse
	err = json.Unmarshal(data, &accountReceived)
	require.NoError(t, err)

	require.Equal(t, accountReceived, account)
}

func requireBodyMatchAccounts(t *testing.T, b *bytes.Buffer, accounts []*accountResponse) {
	data, err := io.ReadAll(b)
	require.NoError(t, err)

	var accountsReceived []db.Account
	err = json.Unmarshal(data, &accountsReceived)
	require.NoError(t, err)

	require.Equal(t, accountsReceived, accounts)
}

func TestGetAccount(t *testing.T) {
	user, _ := createRandomUser(t)
	account := createRandomAccount(user.Username)
	accountResponse := mapAccountToResponse(account)

	testCases := []struct {
		name      string
		accountId int64
		testCaseBase
	}{
		{
			name:      "OK",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						GetAccount(gomock.Any(), gomock.Eq(account.ID)).
						Times(1).
						Return(account, nil)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusOK, recorder.Code)
					requireBodyMatchAccount(t, recorder.Body, accountResponse)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name:      "ErrNoRows",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						GetAccount(gomock.Any(), gomock.Any()).
						Times(1).
						Return(db.Account{}, sql.ErrNoRows)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusNotFound, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name:      "InternalError",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						GetAccount(gomock.Any(), gomock.Any()).
						Times(1).
						Return(db.Account{}, sql.ErrConnDone)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusInternalServerError, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name:      "BadRequest",
			accountId: 0,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().
						GetAccount(gomock.Any(), gomock.Any()).
						Times(0)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
	}

	for i := 0; i < len(testCases); i++ {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			url := "/api/accounts"

			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			runServerTest(t, tc, req)
		})
	}
}

func TestListAccount(t *testing.T) {
	user, _ := createRandomUser(t)

	accountsResponses := []*accountResponse{
		mapAccountToResponse(createRandomAccount(user.Username)),
		mapAccountToResponse(createRandomAccount(user.Username)),
		mapAccountToResponse(createRandomAccount(user.Username)),
	}

	testCases := []struct {
		name string
		testCaseBase
	}{
		{
			name: "OK",
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccounts(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(accountsResponses, nil)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusOK, recorder.Code)
					requireBodyMatchAccounts(t, recorder.Body, accountsResponses)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name: "BadRequest",
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccounts(gomock.Any(), gomock.Any()).
						Times(0)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
		{
			name: "InternalError",
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccounts(gomock.Any(), gomock.Any()).
						Times(1).
						Return([]db.Account{}, sql.ErrTxDone)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusInternalServerError, recorder.Code)
				},
				setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, user.Username)
				},
			},
		},
	}

	for i := 0; i < len(testCases); i++ {
		tc := testCases[i]

		url := "/api/accounts"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		runServerTest(t, tc, req)
	}
}

func TestDeleteAccount(t *testing.T) {
	account := createRandomAccount(util.RandomOwner())

	testCases := []struct {
		name      string
		accountId int64
		testCaseBase
	}{
		{
			name:      "OK",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
					store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(nil)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusOK, recorder.Code)
				}, setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, account.Owner)
				},
			},
		},
		{
			name:      "BadRequest",
			accountId: 0,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(0)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, recorder.Code)
				}, setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, account.Owner)
				},
			},
		},
		{
			name:      "NotFound",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusNotFound, recorder.Code)
				}, setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, account.Owner)
				},
			},
		},
		{
			name:      "Unauthorized",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusUnauthorized, recorder.Code)
				}, setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, util.RandomOwner())
				},
			},
		},
		{
			name:      "Unauthenticated",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusUnauthorized, recorder.Code)
				}, setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {},
			},
		},
		{
			name:      "InternalError",
			accountId: account.ID,
			testCaseBase: testCaseBase{
				buildStubs: func(store *mockdb.MockStore) {
					store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
					store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(sql.ErrConnDone)
				},
				checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusInternalServerError, recorder.Code)
				}, setupAuth: func(t *testing.T, req *http.Request, maker token.Maker) {
					addAuthHeader(t, req, maker, authorizationTypeBearer, account.Owner)
				},
			},
		},
	}

	for i := 0; i < len(testCases); i++ {
		tc := testCases[i]

		url := fmt.Sprintf("/api/accounts/%d", tc.accountId)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		require.NoError(t, err)

		runServerTest(t, tc, req)
	}
}
