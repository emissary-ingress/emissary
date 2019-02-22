.PHONY: echo 

 echo: echo.build-docker echo.clean echo.generate 

 echo.build-docker:
	@docker build -t echo-api-build .

 echo.clean:
	@rm -rf echo-service.pb.go

 echo.generate:	
	@docker run -it -v ${PWD}/:/echo echo-api-build:latest
