# Error Schema Ownership

Currently hard-coded in apispec's Generate function. Should it live in
boiler/respond instead, since that's where the error response convention
is defined? Consider whether a Go struct for error responses would be
valuable — then it could be generated like any other type.
