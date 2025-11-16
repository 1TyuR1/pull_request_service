package errs

type ErrorCode string

const (
	CodeTeamExists  ErrorCode = "TEAM_EXISTS"
	CodePRExists    ErrorCode = "PR_EXISTS"
	CodePRMerged    ErrorCode = "PR_MERGED"
	CodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	CodeNoCandidate ErrorCode = "NO_CANDIDATE"
	CodeNotFound    ErrorCode = "NOT_FOUND"
)

type AppError struct {
	Code ErrorCode
	Msg  string
}

func (e *AppError) Error() string {
	return e.Msg
}

func New(code ErrorCode, msg string) *AppError {
	return &AppError{Code: code, Msg: msg}
}
