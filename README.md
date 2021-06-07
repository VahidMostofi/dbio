# DB IO
A project to simulate multiple read/writes to SQL database. The code can handle live update on the database schema, defined on an external source.
This is achieved by exploiting [Go code generation](https://blog.golang.org/generate) and watching the `type_mappings.json` file. Whenever the file that describes the type mappings is changed, the application re-generate the necessary source codes and re-compile them, then restarts itself inside the Docker container. Note that the Docker container hosting each application **DOES NOT** stop when this happens.

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