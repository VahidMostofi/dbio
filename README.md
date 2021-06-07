# DB IO
A project to simulate multiple read/writes to SQL database. The code can handle live update on the database schema, defined on an external source.
This is achieved by exploiting [Go code generation](https://blog.golang.org/generate) and watching the `type_mappings.json` file. Whenever the file that describes the type mappings is changed, the application re-generate the necessary source codes and re-compile them, then restarts itself inside the Docker container. Note that the Docker container hosting each application **DOES NOT** stop when this happens.

## Features
- `type_mappings.json` defines the list of events and fields of events. While the stack is up and running (in docker-compose), changing the `type_mappings.json` file would result in the system updating itself (without stopping the containers). It updates both the struct definitions and database tables **on the fly**.
- db-writer and db-reader are two separate applications and can **run simultaneously** without any issue.
- db-writer and db-reader each can **scale up or down** without interrupting other instances.

## How to deploy the stack
Use `docker-compose up` to deploy the stack. Use `docker-compose up --build` to force a build before deploying the stack.

## Reader/Writer
  Both applications run inside Docker containers. Entrypoint for both applications are `run-with-reborn.sh` which both generate necessary codes and build them and then runs the reader/writer. In case of `type_mappings.json` changes (that is mapped through volumes in Docker), both reader and writer detect that, and exit with specific exit code =36. If exit code inside `run-with-reborn.sh` is 36, it redo the process (generate, build, run) otherwise exists.

  ### Behaviors:

  - If the `type_mappings.json` is invalid or has errors the application stops running to prevent the system form inserting incorrect values. **This behavior can be easily changed based on the requirements**, for example ignore those errors and using the oldest version.
  - Currently both reader and writer monitor the `type_mappings.json` file and migrate the database, this can be changed so only writer does that.
  - Currently migration is done using `gorm` package but because performance issues resulting from using an ORM, the read and write operations are done using `database\sql` package without any ORM.
  - Only these types are supported now, more types can be supported with contributing with the team provided the problem statement.
    - timestamp
    - int
    - int64
    - bigint
  - Timezone for both applications is set to `TimeZone=UTC` for consistency.
### Reader/Writer environment variables
  - `TYPE_MAPPING_PATH` path to type_mappings file
  - `DB_HOST` database host address
  - `DB_PORT` database port number
  - `DB_USER` database user
  - `DB_PASSWORD` database password
  - `DB_DATABASE` database name
  - `RANDOM_SEED` random seed to use 
  - `CHECK_INTERVAL` how often the reader or writer should check the type_mappings file for changes
  - `READ_INTERVAL` only for the db-reader, how often the reader should read values from the database? examples: `4s`, `100ms`, `5m`, `4s100ms`
  - `WRITE_INTERVAL` only for the db-writer, how often the writer should read values from the database? examples: `4s`, `100ms`, `5m`, `4s100ms`

### How to scale
When the stack is up, use `docker-compose scale dbwriter=x dbreader=y` to set the number of instances of `dbwriter` to `x` and `dbreader` to `y`. The scaling can be done in both direction without affecting other instances.

### Database
- Use `database.env` to configure the database. The existing version is a sample to make `docker-compose` work. 
- Port mapping in `docker-compose` file is due debugging and developing, you can remove that if you don't want to connect to the database from your local machine.
- Currently no volume is mounted for the database in docker-compose, add the following lines to have persistent data (replace `$HOME/data` with your desired path on the local system).
  ```
  volumes:
      - $HOME/data:/var/lib/postgresql/data/ 
  ```
- While there is no volume mapping by default for the database, if you don't use `docker-compose down` to stop the stack, the database container will probably have the data next time. Be safe and use `docker-compose down` to stop everything gracefully.
- As this is just a sample, `sslmode` is disable.

## Items that need to be improved
- Random value generators returns random values only based on the type, this can be improved to work based on another configuration file to generate random values based on specific event and field (`Time` is an exception and random values for this field follow a patter to makes the simulation feels more natural).
## Initial Requirements
Two applications are needed, DB Writer and DB Reader. These should be separate applications and are expected to be able to run simultaneously.
    
- **DB Writer**:
  - This application’s role is to process events and store them in an SQL database.
  - It should insert 1 new record into the database every `t` seconds with random values.
  - The structure of the various events is defined in a source, in this example the `type_mapping.json`.
  - The application should be able to handle additions of events and/or fields in this file.
  - The application must read the json and update the database schema accordingly if applicable.
  - Existing data previously created with a different type mappings should be preserved.
  - Any identical events that get processed should produce a single record in the database (no duplicate rows).
  - Should be easily scalable (having multiple instances).

- **DB Reader**:
  - This application must query the database `f` times per second.
  - Each query should be for a random event type and filter a random time frame between a start and end time.
  - Should be easily scalable (having multiple instances).

- General assessment factors:
  - The applications act as described. Correct Behavior.No crashes.
  - Database performance. Low RAM and CPU usage. Fast reads.
  - Readable. The codebase is easy to follow with comments where appropriate.