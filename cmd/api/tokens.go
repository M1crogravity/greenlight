package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/m1crogravity/greenlight/internal/data"
	"github.com/m1crogravity/greenlight/internal/validator"
	"github.com/pascaldekloe/jwt"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	var claims jwt.Claims
	claims.Subject = strconv.FormatInt(user.ID, 10)
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(time.Now().Add(24 * time.Hour))
	claims.Issuer = "greenlight.d-naidenko"
	claims.Audiences = []string{"greenlight.d-naidenko"}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(app.config.jwt.secret))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJson(w, http.StatusCreated, envelope{"authentication_token": string(jwtBytes)}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if !user.Activated {
		v.AddError("email", "user account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 45*time.Minute, data.ScopePasswordReset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]any{
			"passwordResetToken": token.Plaintext,
		}

		err = app.mailer.Send(user.Email, "token_password_reset.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	env := envelope{"message": "an email will be sent to you containing password reset instructions"}

	err = app.writeJson(w, http.StatusAccepted, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
