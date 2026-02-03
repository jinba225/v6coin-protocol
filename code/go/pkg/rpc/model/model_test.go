// Package model provides tests for RPC data models.
package model

import (
	"testing"
)

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		code        ErrorCode
		message     string
		wantSuccess bool
	}{
		{
			name:        "error response with message",
			code:        CodeInvalidParameter,
			message:     "Invalid parameter",
			wantSuccess: false,
		},
		{
			name:        "error response without message",
			code:        CodeBlockNotFound,
			message:     "",
			wantSuccess: false,
		},
		{
			name:        "unknown error code",
			code:        99999,
			message:     "Custom error",
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ErrorResponse(tt.code, tt.message)
			if resp.Success != tt.wantSuccess {
				t.Errorf("ErrorResponse().Success = %v, want %v", resp.Success, tt.wantSuccess)
			}
			if tt.message != "" && resp.Message != tt.message {
				t.Errorf("ErrorResponse().Message = %v, want %v", resp.Message, tt.message)
			}
		})
	}
}

func TestSuccessResponse(t *testing.T) {
	tests := []struct {
		name       string
		data       interface{}
		wantSuccess bool
	}{
		{
			name:       "success response with string data",
			data:       "ok",
			wantSuccess: true,
		},
		{
			name:       "success response with nil data",
			data:       nil,
			wantSuccess: true,
		},
		{
			name:       "success response with struct",
			data:       struct{ Name string }{Name: "test"},
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := SuccessResponse(tt.data)
			if resp.Success != tt.wantSuccess {
				t.Errorf("SuccessResponse().Success = %v, want %v", resp.Success, tt.wantSuccess)
			}
			if tt.data == nil && resp.Data != nil {
				t.Errorf("SuccessResponse().Data should be nil")
			}
			if tt.data != nil && resp.Data == nil {
				t.Errorf("SuccessResponse().Data should not be nil")
			}
			if resp.Message != GetErrorMessage(CodeSuccess) {
				t.Errorf("SuccessResponse().Message = %v, want %v", resp.Message, GetErrorMessage(CodeSuccess))
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want string
	}{
		{"success code", CodeSuccess, "Success"},
		{"invalid request", CodeInvalidRequest, "Invalid request format or parameters"},
		{"missing parameter", CodeMissingParameter, "Required parameter is missing"},
		{"invalid parameter", CodeInvalidParameter, "Parameter has invalid value or format"},
		{"unauthorized", CodeUnauthorized, "Authentication required"},
		{"block not found", CodeBlockNotFound, "Block not found"},
		{"transaction not found", CodeTransactionNotFound, "Transaction not found"},
		{"account not found", CodeAccountNotFound, "Account not found"},
		{"node not found", CodeNodeNotFound, "Node not found"},
		{"network error", CodeNetworkError, "Network connection error"},
		{"unknown error", ErrorCode(99999), "Unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetErrorMessage(tt.code)
			if got != tt.want {
				t.Errorf("GetErrorMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name  string
		code  ErrorCode
		value int
	}{
		{"success code", CodeSuccess, 0},
		{"invalid request", CodeInvalidRequest, 1000},
		{"missing parameter", CodeMissingParameter, 1001},
		{"invalid parameter", CodeInvalidParameter, 1002},
		{"unauthorized", CodeUnauthorized, 1003},
		{"forbidden", CodeForbidden, 1004},
		{"rate limit", CodeRateLimitExceeded, 1005},
		{"service unavailable", CodeServiceUnavailable, 1006},
		{"block not found", CodeBlockNotFound, 10001},
		{"invalid block height", CodeInvalidBlockHeight, 10002},
		{"invalid block hash", CodeInvalidBlockHash, 10003},
		{"block validation", CodeBlockValidation, 10004},
		{"invalid merkle root", CodeInvalidMerkleRoot, 10005},
		{"invalid parent hash", CodeInvalidParentHash, 10006},
		{"transaction not found", CodeTransactionNotFound, 20001},
		{"invalid transaction type", CodeInvalidTransactionType, 20002},
		{"invalid transaction", CodeInvalidTransaction, 20003},
		{"insufficient balance", CodeInsufficientBalance, 20004},
		{"invalid nonce", CodeInvalidNonce, 20005},
		{"invalid signature", CodeInvalidSignature, 20006},
		{"duplicate transaction", CodeDuplicateTransaction, 20007},
		{"transaction too old", CodeTransactionTooOld, 20008},
		{"invalid fee", CodeInvalidFee, 20009},
		{"transaction pool full", CodeTransactionPoolFull, 20010},
		{"account not found", CodeAccountNotFound, 30001},
		{"invalid address", CodeInvalidAddress, 30002},
		{"address mismatch", CodeAddressMismatch, 30003},
		{"node not found", CodeNodeNotFound, 40001},
		{"invalid node id", CodeInvalidNodeID, 40002},
		{"not validator", CodeNotValidator, 40003},
		{"validator not found", CodeValidatorNotFound, 40004},
		{"network error", CodeNetworkError, 50001},
		{"peer unavailable", CodePeerUnavailable, 50002},
		{"sync in progress", CodeSyncInProgress, 50003},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.code) != tt.value {
				t.Errorf("ErrorCode value = %v, want %v", tt.code, tt.value)
			}
		})
	}
}
