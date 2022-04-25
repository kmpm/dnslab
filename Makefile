


.PHONY: build
build: windows linux
	

.PHONY: windows
windows: export GOOS=windows
windows: 
	go build ./

.PHONY: linux
linux: export GOOS=linux
linux: 
	go build ./


.PHONY: loopia
loopia:
	.\dnslab.exe -server ns1.loopia.se  _acme-challenge.test1.kapi.se
	.\dnslab.exe -edns0 -server ns1.loopia.se  _acme-challenge.test1.kapi.se

.PHONY: glesys
glesys:
	.\dnslab.exe  _acme-challenge.test1.kapi.kmpm.dev
	.\dnslab.exe -edns0  _acme-challenge.test1.kapi.kmpm.dev