package redis

// Errors that may be raised during config parsing.
type RedisError string

func (e RedisError) Error() string {
	return string(e)
}

// Interface for a redis connection pool.
type Pool interface {
	// Get a connection from the pool. Call Put() on the connection when done.
	// Throws RedisError if a connection can not be obtained.
	Get() Connection

	// Put a connection back into the pool.
	// @param c supplies the connection to put back.
	Put(c Connection)
}

// Interface for a redis connection.
type Connection interface {
	// Append a command onto the pipeline queue.
	// @param command supplies the command to append.
	// @param args supplies the additional arguments.
	PipeAppend(command string, args ...interface{})

	// Execute the pipeline queue and wait for a response.
	// @return a response object.
	// Throws a RedisError if there was an error fetching the response.
	PipeResponse() Response
}

// Interface for a redis response.
type Response interface {
	// @return the response as an integer.
	// Throws a RedisError if the response is not convertable to an integer.
	Int() int64
}
