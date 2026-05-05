 1. Critical: live secrets are hardcoded in source.
     internal/application/application.go:65 hardcodes a Postgres DSN/password, internal/application/application.go:77 and internal/application/application.go:78 hardcode
     SMTP credentials, and internal/application/application.go:89 plus internal/application/application.go:91 hardcode Google OAuth client credentials. Treat these as
     compromised if this repo has ever left your machine. Move them to env/config, remove defaults for production secrets, and rotate the exposed credentials.
  2. High: auth tokens are logged through request URLs.
     cmd/api/middleware.go:25 logs r.URL.String(). That captures /auth/activate?token=... from handlers/authHandler/routeHandlers.go:96, and also OIDC code/state on /auth/
     google/redirect from handlers/authHandler/routeHandlers.go:246. Activation tokens and authorization codes should not appear in app logs. Log path only, or redact
     sensitive query params.
  3. High: unactivated email/password users can log in.
     handlers/authHandler/routeHandlers.go:190 has a TODO for activation checking, but handlers/authHandler/routeHandlers.go:200 issues a session after password match.
     Registration creates activation tokens, but the login flow does not enforce user.Activated. Add an activation gate before app.Login.
  4. Medium: login leaks user existence and has a nil-password edge case.
     handlers/authHandler/routeHandlers.go:182 returns "email not registered", while bad passwords return "invalid credentials" at handlers/authHandler/
     routeHandlers.go:196. That enables account enumeration. Also, OIDC-created users can have password_hash = NULL; handlers/authHandler/routeHandlers.go:191 passes that
     into bcrypt and likely returns a 500, creating another enumeration signal. Return one generic login error for all auth failures and handle nil password hashes as non-
     matches.
  5. Medium: activation is a state-changing GET with token in the URL.
     handlers/authHandler/routes.go:29 exposes GET /auth/activate, and handlers/authHandler/routeHandlers.go:120 through handlers/authHandler/routeHandlers.go:152 activates
     the account and logs the user in. Query tokens are prone to logging, browser history, proxies, and referrer leakage. Prefer a frontend landing page that POSTs the
     token, or at least make logging/redaction airtight.
  6. Medium: OIDC JWT validation is stricter than nothing, but still incomplete/fragile.
     internal/application/OIDC/google/oidc.go:274 allows an empty alg instead of requiring exactly RS256. internal/application/OIDC/google/oidc.go:118 fetches JWKS once at
     startup and never refreshes, so Google key rotation can break login. Also, discovery/JWKS/token responses are decoded without checking HTTP status at internal/
     application/OIDC/google/oidc.go:90, internal/application/OIDC/google/oidc.go:118, and handlers/authHandler/routeHandlers.go:278. Consider using a maintained OIDC
     verifier library instead of hand-rolling this.
  7. Low/Medium: OIDC state/nonce storage is unbounded per session.
     handlers/authHandler/helper.go:18 notes this, and handlers/authHandler/helper.go:34 appends forever. A client can repeatedly hit /auth/google/oidcURL and grow session
     data in the DB. Cap entries and expire old ones.
