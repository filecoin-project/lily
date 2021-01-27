package ffi

func Hash(message Message) Digest {
	return Digest{}
}

func Verify(signature *Signature, digests []Digest, publicKeys []PublicKey) bool {
	return true
}

func HashVerify(signature *Signature, messages []Message, publicKeys []PublicKey) bool {
	return true
}

func Aggregate(signatures []Signature) *Signature {
	var s Signature
	return &s
}

func PrivateKeyGenerate() PrivateKey {
	return PrivateKey{}
}

func PrivateKeyGenerateWithSeed(seed PrivateKeyGenSeed) PrivateKey {
	return PrivateKey{}
}

func PrivateKeySign(privateKey PrivateKey, message Message) *Signature {
	var s Signature
	return &s
}

func PrivateKeyPublicKey(privateKey PrivateKey) PublicKey {
	return PublicKey{}
}

func CreateZeroSignature() Signature {
	return Signature{}
}
