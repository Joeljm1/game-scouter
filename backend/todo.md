# TODO
1) [X] Session lazy saving (save data only when session data is edited)
2)  [ ] maybe jwt for before login??
3) [X] clear cache val after some time and be concurrent safe
4) [ ] Fix error handling of old code ie use custom error types
5) [ ] Add cache for token as it can be expensive with it being a jsonb
6) [X] Add context to all db operation from request
7) [ ] session orginal method not great, change it and cache it 
8) [ ] add session data to context only when required (lazy loading)
9) [ ] *Buffered loggin would be great*
10) [ ] Clean old token data and old session data for a user ( may be do them in db layer??)


# Passsword changes to all oidc

1) [X] Make password a *string
2) [X] Make password a nullable in table
3) [X] Make it compile
4) [ ] Check for security issues
5) [ ] Changes to token table to add type of auth done so that i can check activated user field only if token type is email
6) [ ] Or may be add a scope to token instead of column omgg!! that easier
