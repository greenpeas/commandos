# Commandos - микросервис для работы с командами

### Settings for private repo modules

Execute:

`go env -w GOPRIVATE=gitlab.kvant.online/*`

and 

`git config --global url."ssh://git@192.168.158.80".insteadOf "https://gitlab.kvant.online"`


### Update contracts
`go get -u gitlab.kvant.online/seal/grpc-contracts`

### Update command validator
`go get -u gitlab.kvant.online/seal/driver`