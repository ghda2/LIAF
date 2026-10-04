package builtins

func init() {
	// ---------------------------------------------------------
	// CLI & Process
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "cli-args",
		Category:     "system",
		Arity:        ExactArity(0),
		Return:       "(list str)",
		GoCall:       "rt.CliArgs",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "exit-process",
		Category:     "system",
		Arity:        ExactArity(1),
		Params:       []string{"int"},
		Effects:      []string{"io"},
		Return:       "void",
		GoCall:       "rt.ExitProcess",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Filesystem
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "fs-mkdir-all",
		Category:     "fs",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "(result bool str)",
		GoCall:       "rt.FsMkdirAll",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "fs-remove-all",
		Category:     "fs",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "(result bool str)",
		GoCall:       "rt.FsRemoveAll",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "fs-list-dir",
		Category:     "fs",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "(result (list str) str)",
		GoCall:       "rt.FsListDir",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "fs-is-dir",
		Category:     "fs",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "bool",
		GoCall:       "rt.FsIsDir",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "fs-file-size",
		Category:     "fs",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "(result int str)",
		GoCall:       "rt.FsFileSize",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Subprocesses
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "os-exec",
		Category:     "system",
		Arity:        ExactArity(2),
		Params:       []string{"str", "(list str)"},
		Effects:      []string{"io"},
		Return:       "(result str str)",
		GoCall:       "rt.OsExec",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "os-exec-status",
		Category:     "system",
		Arity:        ExactArity(2),
		Params:       []string{"str", "(list str)"},
		Effects:      []string{"io"},
		Return:       "int",
		GoCall:       "rt.OsExecStatus",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// CSV
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "csv-parse",
		Category:     "csv",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result (list (list str)) str)",
		GoCall:       "rt.CsvParse",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "csv-encode",
		Category:     "csv",
		Arity:        ExactArity(1),
		Params:       []string{"(list (list str))"},
		Return:       "str",
		GoCall:       "rt.CsvEncode",
		UnsupportedC: true,
	})
}
