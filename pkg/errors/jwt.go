package errors

var (
	// ErrInvalidPrivKey indicates that the given private key is invalid
	ErrInvalidPrivKey = New("Private key invalid")

	// ErrInvalidPubKey indicates the the given public key is invalid
	ErrInvalidPubKey = New("Public key invalid")

	// ErrFailedTokenCreation indicates JWT Token failed to create, reason unknown
	ErrFailedTokenCreation = New("Failed to create JWT Token")

	// ErrInvalidAuthHeader indicates invalid Authorization header
	ErrInvalidAuthHeader = New("Authorization header invalid")

	// ErrMissingToken indicates the jwt token is not present
	ErrMissingToken = New("Missing Access Token")

	// ErrInvalidSigningAlgorithm indicates signing algorithm is invalid, needs to be HS256, HS384, HS512, RS256, RS384 or RS512
	ErrInvalidSigningAlgorithm = New("Invalid signing algorithm")

	// ErrInvalidJWTToken indicates JWT token is invalid.
	ErrInvalidJWTToken = New("Invalid token")

	// ErrExpiredToken indicates JWT token has expired.
	ErrExpiredToken = New("Token has expired")

	// ErrMissingUserData  indicates user data missing while setting claim  in JWT token.
	ErrMissingUserData = New("Missing user data while creating token")

	// ErrMissingUserID indicates user_id claim missing in JWT token.
	ErrMissingUserID = New("Missing user_id in Token")

	// ErrMissingJTI indicates jti claim missing in JWT token.
	ErrMissingJTI = New("Missing jti in Token")

	// ErrMissingGitProvider indicates git_provider claim missing in JWT token.
	ErrMissingGitProvider = New("Missing git_provider in Token")

	// ErrMissingUserName indicates username claim missing in JWT token.
	ErrMissingUserName = New("Missing username in Token")
)
