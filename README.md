# project-brian-api-test
Integration tests for project-brian

Get the code:

```
go get github.com/ONSdigital/project-brian-api-test
```

### Set up

**Before you can run the tests**

- Unzip the `/resources/outputs.zip`:

    ```
    cd resources
    unzip outputs.zip -d outputs
    ```
- Run [project-brian](https://github.com/ONSdigital/project-brian)

### Running the tests

```
go test -v -failfast
```

While you can use `goconvey` to run the tests it is discouraged. The number of assertions to verify the larger `.csdb` 
files do not play nicely with the browser UI.  

### About the tests

The tests make a `POST` request to `Services/ConvertCSDB` for each `.csdb` files under `/resources/inputs`. The 
response is captured and compared to the expected response under `/resources/outputs`

For example:

```
HTTP: POST 
    /resources/inputs/ott.csdb -> Services/ConvertCSDB
Compare:
    response.body -> /resources/outputs/ott-csdb.json
```

:warning: **IMPORTANT:** :warning: The inputs and expected outputs a tightly coupled. If you modify an input file to 
contains different data you will have to recreate the expected response json and add replace the current 
`/resources/outputs.zip`. 