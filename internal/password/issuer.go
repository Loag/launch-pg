package password

// Secret is a freshly issued password: Plain goes to the user once,
// Verifier goes to the server.
type Secret struct {
	Plain    string
	Verifier string
}

// Issuer generates and hashes new passwords.
type Issuer struct {
	generator Generator
	hasher    Hasher
}

func NewIssuer(generator Generator, hasher Hasher) *Issuer {
	return &Issuer{generator: generator, hasher: hasher}
}

func (i *Issuer) Issue() (Secret, error) {
	plain, err := i.generator.Generate()
	if err != nil {
		return Secret{}, err
	}
	verifier, err := i.hasher.Hash(plain)
	if err != nil {
		return Secret{}, err
	}
	return Secret{Plain: plain, Verifier: verifier}, nil
}
