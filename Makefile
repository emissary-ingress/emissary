all: ambassador

.ALWAYS:

ambassador: .ALWAYS
	docker build -t dwflynn/ambassador:0.1.2 ambassador
	docker push dwflynn/ambassador:0.1.2

