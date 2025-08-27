package authentication

import (
	"net/http"

	resterr "github.com/zdnscloud/gorest/error"
	"github.com/gsmlg-opt/gaocloud/pkg/authentication/cas"
	"github.com/gsmlg-opt/gaocloud/pkg/authentication/jwt"
	"github.com/gsmlg-opt/gaocloud/pkg/types"
)

type Authenticator struct {
	JwtAuth *jwt.Authenticator
	CasAuth *cas.Authenticator
}

func New(casServer string) (*Authenticator, error) {
	jwtAuth, err := jwt.NewAuthenticator()
	if err != nil {
		return nil, err
	}

	auth := &Authenticator{
		JwtAuth: jwtAuth,
	}

	if casServer != "" {
		casAuth, err := cas.NewAuthenticator(casServer)
		if err != nil {
			return nil, err
		}
		auth.CasAuth = casAuth
	}
	return auth, nil
}

func (a *Authenticator) Authenticate(w http.ResponseWriter, req *http.Request) (string, *resterr.APIError) {
	user, err := a.JwtAuth.Authenticate(w, req)
	if err != nil {
		return "", err
	} else if user != "" {
		return user, nil
	}

	if a.CasAuth == nil {
		return "", nil
	} else {
		user, err := a.CasAuth.Authenticate(w, req)
		if err == nil && user != "" {
			if !a.JwtAuth.HasUser(user) {
				newUser := &types.User{Name: user}
				newUser.SetID(user)
				a.JwtAuth.AddUser(newUser)
			}
		}
		return user, err
	}
}
