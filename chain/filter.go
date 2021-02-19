package chain

func NewAddressFilter(addr string) *AddressFilter {
	return &AddressFilter{address: addr}
}

type AddressFilter struct {
	address string
}

func (f *AddressFilter) Allow(addr string) bool {
	return f.address == addr
}
