package auth_test

import (
	"testing"
	"time"

	// "time"

	"github.com/brinwiththevlin/Chirpy-http-server/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	// "github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		password string
		wantErr  bool
	}{
		{name: "valid password", password: "password123", wantErr: false},
		{name: "empty password", password: "", wantErr: false},
		{name: "short password", password: "abc", wantErr: false},
		{name: "long password", password: "aVeryLongPasswordThatExceedsTheMaximumLength", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := auth.HashPassword(tt.password)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("HashPassword() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("HashPassword() succeeded unexpectedly")
			}
		})
	}
}



func TestCheckPasswordHash(t *testing.T) {
    // setup: generate a valid hash for a known password
    password := "s3cr3t!"
    validHash, err := auth.HashPassword(password)
    if err != nil {
        t.Fatalf("setup: HashPassword(%q) failed: %v", password, err)
    }

    tests := []struct {
        name     string
        hash     string
        password string
        wantErr  bool
    }{
        {
            name:     "matching password",
            hash:     validHash,
            password: password,
            wantErr:  false,
        },
        {
            name:     "non-matching password",
            hash:     validHash,
            password: "nope",
            wantErr:  true,
        },
        {
            name:     "empty hash",
            hash:     "",
            password: password,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := auth.CheckPasswordHash(tt.hash, tt.password)
            if (err != nil) != tt.wantErr {
                t.Fatalf("CheckPasswordHash(%q, %q) error = %v, wantErr %v",
                    tt.hash, tt.password, err, tt.wantErr)
            }
        })
    }
}


func TestMakeJWT(t *testing.T) {
	tests := []struct {
		name         string
		userID       uuid.UUID
		tokenSecret  string
		expiresIn    time.Duration
		parseSecret  string   // what key to use when parsing
		wantErrParse bool     // expect parse to fail?
	}{
		{
			name:         "valid token",
			userID:       uuid.New(),
			tokenSecret:  "super-secret",
			expiresIn:    2 * time.Hour,
			parseSecret:  "super-secret",
			wantErrParse: false,
		},
		{
			name:         "invalid secret",
			userID:       uuid.New(),
			tokenSecret:  "secret-A",
			expiresIn:    30 * time.Minute,
			parseSecret:  "different-secret",
			wantErrParse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1) generate
			tokenStr, err := auth.MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)
			if err != nil {
				t.Fatalf("MakeJWT() error = %v", err)
			}
			if tokenStr == "" {
				t.Fatal("MakeJWT() returned empty token")
			}

			// 2) parse with the given parseSecret
			claims := &jwt.RegisteredClaims{}
			parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
				return []byte(tt.parseSecret), nil
			})
			if tt.wantErrParse {
				if err == nil && parsed.Valid {
					t.Fatal("expected parse to fail, but it succeeded")
				}
				return
			}
			// parse should succeed
			if err != nil {
				t.Fatalf("ParseWithClaims() error = %v", err)
			}
			if !parsed.Valid {
				t.Fatal("parsed token is not valid")
			}

			// 3) verify the subject matches our userID
			if claims.Subject != tt.userID.String() {
				t.Errorf("claims.Subject = %q; want %q", claims.Subject, tt.userID.String())
			}

			// 4) verify expiration is roughly now + expiresIn
			if claims.ExpiresAt == nil {
				t.Error("ExpiresAt is nil")
			} else {
				remaining := time.Until(claims.ExpiresAt.Time)
				if remaining < tt.expiresIn-time.Minute || remaining > tt.expiresIn+time.Minute {
					t.Errorf("token expiry = %v from now; want ~%v", remaining, tt.expiresIn)
				}
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	validToken, _:= auth.MakeJWT(userID, "secret", time.Hour)
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		tokenString string
		tokenSecret string
		wantUserID        uuid.UUID
		wantErr     bool
	}{
		{
			name:        "Valid token",
			tokenString: validToken,
			tokenSecret: "secret",
			wantUserID:  userID,
			wantErr:     false,
		},
		{
			name:        "Invalid token",
			tokenString: "invalid.token.string",
			tokenSecret: "secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "Wrong secret",
			tokenString: validToken,
			tokenSecret: "wrong_secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := auth.ValidateJWT(tt.tokenString, tt.tokenSecret)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ValidateJWT() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ValidateJWT() succeeded unexpectedly")
			}
			if got != tt.wantUserID{
				t.Errorf("ValidateJWT() = %v, want %v", got, tt.wantUserID)
			}
		})
	}
}

