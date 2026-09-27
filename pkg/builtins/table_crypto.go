package builtins

func init() {
	// ---------------------------------------------------------
	// Crypto
	// ---------------------------------------------------------
	// sha256 e hmac-sha256 recebem a codificacao como literal ("hex" ou
	// "base64url"), conferido pelo checker como o driver de db-connect.
	Register(&Builtin{
		Name:         "sha256",
		Category:     "crypto",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Return:       "str",
		GoCall:       "rt.SHA256",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "hmac-sha256",
		Category:     "crypto",
		Arity:        ExactArity(3),
		Params:       []string{"str", "str", "str"},
		Return:       "str",
		GoCall:       "rt.HMACSHA256",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "secure-eq",
		Category:     "crypto",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Return:       "bool",
		GoCall:       "rt.SecureEq",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "base64url-encode",
		Category:     "crypto",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "str",
		GoCall:       "rt.Base64URLEncode",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "base64url-decode",
		Category:     "crypto",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.Base64URLDecode",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "random-token",
		Category:     "crypto",
		Arity:        ExactArity(0),
		Effects:      []string{"rand"},
		Return:       "str",
		GoCall:       "rt.RandomToken",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "password-hash",
		Category:     "crypto",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"rand"},
		Return:       "str",
		GoCall:       "rt.PasswordHash",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "password-verify",
		Category:     "crypto",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Return:       "bool",
		GoCall:       "rt.PasswordVerify",
		UnsupportedC: true,
	})
}
