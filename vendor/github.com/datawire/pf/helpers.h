static u_int16_t htons_f(u_int16_t hostshort) {
	return htons(hostshort);
}

static u_int16_t ntohs_f(u_int16_t netshort) {
	return ntohs(netshort);
}

static void set_addr_port(struct pf_rule_addr *addr, int index, u_int16_t value) {
	addr->xport.range.port[index] = value;
}
static u_int16_t get_addr_port(struct pf_rule_addr *addr, int index) {
	return addr->xport.range.port[index];
}

static void set_addr_port_op(struct pf_rule_addr *addr, u_int8_t op) {
	addr->xport.range.op = op;
}
static u_int8_t get_addr_port_op(struct pf_rule_addr *addr) {
	return addr->xport.range.op;
}

static void set_natlook_sport(struct pfioc_natlook *pnl, u_int16_t port) {
	pnl->sxport.port = port;
}

static void set_natlook_dport(struct pfioc_natlook *pnl, u_int16_t port) {
	pnl->dxport.port = port;
}

static u_int16_t get_natlook_rdport(struct pfioc_natlook *pnl) {
	return pnl->rdxport.port;
}

struct pton_addr {
	char value[INET_ADDRSTRLEN];
};

static char *get_pton_addr(struct pton_addr *addr) {
	return addr->value;
}
