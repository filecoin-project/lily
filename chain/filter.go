package chain

func NewAddressFilter(addr string) *AddressFilter {
	return &AddressFilter{address: addr}
}

type AddressFilter struct {
	address string
}

func (f *AddressFilter) Allow(addr string) bool {
	if f.address == addr {
		return true
	}
	return false
}
