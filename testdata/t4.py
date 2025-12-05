# Do not reformat this one, see gotags_test.go for instructions.  There may be literal tabs in the
# comments.

#builtin-etags

import zappa

def fib(n): #D |def fib|
    if n < 2:
        return n
    return fib(n-1) + fib(n-2)

    async def effer(n): #D |    async def effer|
        await fib(10)

class MyClass: #D |class MyClass|
    def operate(n):  #D |    def operate|
        return n + 1

    def stopit():  #D |    def stopit|
        return 0
