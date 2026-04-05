package docs

import _ "embed"

//go:embed data/operators.json
var operatorsJSON []byte

//go:embed data/python_api.json
var pythonAPIJSON []byte
