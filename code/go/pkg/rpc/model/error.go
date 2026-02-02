// Package model provides data models for the RPC API.
package model

// ErrorCode represents the error code type.
type ErrorCode int

// Error codes following the RPC API specification.
// Error code ranges:
//
//	0: Success
//	1-9999: Client errors (parameter errors, format errors, etc.)
//	10000-19999: Block-related errors
//	20000-29999: Transaction-related errors
//	30000-39999: Account-related errors
//	40000-49999: Node-related errors
//	50000-59999: Network-related errors
const (
	// Success (0)
	CodeSuccess ErrorCode = 0

	// Client Errors (1-9999)
	CodeInvalidRequest     ErrorCode = 1000 // Invalid request format or parameters
	CodeMissingParameter   ErrorCode = 1001 // Required parameter is missing
	CodeInvalidParameter   ErrorCode = 1002 // Parameter has invalid value or format
	CodeUnauthorized       ErrorCode = 1003 // Authentication required
	CodeForbidden          ErrorCode = 1004 // Access denied
	CodeRateLimitExceeded  ErrorCode = 1005 // Rate limit exceeded
	CodeServiceUnavailable ErrorCode = 1006 // Service temporarily unavailable

	// Block-related Errors (10000-19999)
	CodeBlockNotFound      ErrorCode = 10001 // Block not found
	CodeInvalidBlockHeight ErrorCode = 10002 // Invalid block height
	CodeInvalidBlockHash   ErrorCode = 10003 // Invalid block hash
	CodeBlockValidation    ErrorCode = 10004 // Block validation failed
	CodeInvalidMerkleRoot  ErrorCode = 10005 // Invalid Merkle root
	CodeInvalidParentHash  ErrorCode = 10006 // Invalid parent block hash

	// Transaction-related Errors (20000-29999)
	CodeTransactionNotFound    ErrorCode = 20001 // Transaction not found
	CodeInvalidTransactionType ErrorCode = 20002 // Invalid transaction type
	CodeInvalidTransaction     ErrorCode = 20003 // Transaction validation failed
	CodeInsufficientBalance    ErrorCode = 20004 // Insufficient balance
	CodeInvalidNonce           ErrorCode = 20005 // Invalid nonce
	CodeInvalidSignature       ErrorCode = 20006 // Invalid signature
	CodeDuplicateTransaction   ErrorCode = 20007 // Duplicate transaction
	CodeTransactionTooOld      ErrorCode = 20008 // Transaction too old
	CodeInvalidFee             ErrorCode = 20009 // Invalid transaction fee
	CodeTransactionPoolFull    ErrorCode = 20010 // Transaction pool is full

	// Account-related Errors (30000-39999)
	CodeAccountNotFound ErrorCode = 30001 // Account not found
	CodeInvalidAddress  ErrorCode = 30002 // Invalid IPv6 address format
	CodeAddressMismatch ErrorCode = 30003 // Address does not match expected format

	// Node-related Errors (40000-49999)
	CodeNodeNotFound      ErrorCode = 40001 // Node not found
	CodeInvalidNodeID     ErrorCode = 40002 // Invalid node ID
	CodeNotValidator      ErrorCode = 40003 // Node is not a validator
	CodeValidatorNotFound ErrorCode = 40004 // Validator not found

	// Network-related Errors (50000-59999)
	CodeNetworkError    ErrorCode = 50001 // Network connection error
	CodePeerUnavailable ErrorCode = 50002 // Peer node unavailable
	CodeSyncInProgress  ErrorCode = 50003 // Blockchain sync in progress
)

// ErrorMessages maps error codes to human-readable messages.
var ErrorMessages = map[ErrorCode]string{
	CodeSuccess: "Success",

	// Client Errors
	CodeInvalidRequest:     "Invalid request format or parameters",
	CodeMissingParameter:   "Required parameter is missing",
	CodeInvalidParameter:   "Parameter has invalid value or format",
	CodeUnauthorized:       "Authentication required",
	CodeForbidden:          "Access denied",
	CodeRateLimitExceeded:  "Rate limit exceeded",
	CodeServiceUnavailable: "Service temporarily unavailable",

	// Block Errors
	CodeBlockNotFound:      "Block not found",
	CodeInvalidBlockHeight: "Invalid block height",
	CodeInvalidBlockHash:   "Invalid block hash",
	CodeBlockValidation:    "Block validation failed",
	CodeInvalidMerkleRoot:  "Invalid Merkle root",
	CodeInvalidParentHash:  "Invalid parent block hash",

	// Transaction Errors
	CodeTransactionNotFound:    "Transaction not found",
	CodeInvalidTransactionType: "Invalid transaction type",
	CodeInvalidTransaction:     "Transaction validation failed",
	CodeInsufficientBalance:    "Insufficient balance",
	CodeInvalidNonce:           "Invalid nonce",
	CodeInvalidSignature:       "Invalid signature",
	CodeDuplicateTransaction:   "Duplicate transaction",
	CodeTransactionTooOld:      "Transaction too old",
	CodeInvalidFee:             "Invalid transaction fee",
	CodeTransactionPoolFull:    "Transaction pool is full",

	// Account Errors
	CodeAccountNotFound: "Account not found",
	CodeInvalidAddress:  "Invalid IPv6 address format",
	CodeAddressMismatch: "Address does not match expected format",

	// Node Errors
	CodeNodeNotFound:      "Node not found",
	CodeInvalidNodeID:     "Invalid node ID",
	CodeNotValidator:      "Node is not a validator",
	CodeValidatorNotFound: "Validator not found",

	// Network Errors
	CodeNetworkError:    "Network connection error",
	CodePeerUnavailable: "Peer node unavailable",
	CodeSyncInProgress:  "Blockchain sync in progress",
}

// GetErrorMessage returns the human-readable message for an error code.
func GetErrorMessage(code ErrorCode) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "Unknown error"
}

// ErrorResponse creates a standard error response.
func ErrorResponse(code ErrorCode, message string) *Response {
	if message == "" {
		message = GetErrorMessage(code)
	}
	return &Response{
		Success: false,
		Error:   string(rune(code)),
		Message: message,
	}
}

// SuccessResponse creates a standard success response.
func SuccessResponse(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Message: GetErrorMessage(CodeSuccess),
	}
}
