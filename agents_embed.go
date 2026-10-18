package liaf

import "embed"

// AgentsFS embute as skills de agentes de IA (.agents/skills) no liafc, para
// que `liafc agents` as grave em qualquer projeto sem clone do repositório.
//
//go:embed .agents/skills
var AgentsFS embed.FS
