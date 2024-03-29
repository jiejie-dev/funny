# funny lang

A funny language interpreter written in golang.

It begins just for fun.

## Target

To make creating dsl easily based on funny.

- apitest dsl
- api declare dsl

## Installation

```console
go install github.com/jerloo/funny/cmd/funny@latest
```

## Usage

```javascript

// funny.fun
// author: jerloo@gmail.com
// github: https://github.com/jerloo/funny

echoln('define a varible value 1')
a = 1

echoln('define b varible value 2')
b = 2

echoln('define c varible value a ')
c = a

echoln('a, b, c values: ')
echoln('a = ', a,', b = ',  b, ', c = ', c)

echoln('assert c equels 1')
assert(c == 1)

d = c + b

echoln('assert (d = c + b) === ', d)
assert(d == 3)

echoln('define a function ')
echoln('minus(a, b) {')
echoln('  return b - a')
echoln('}')

minus(a, b) {
  return b - a
}

e = minus(a, b)
echoln('minus(a, b) === ', e)
assert(e == 1)

if a > 0 {
  echoln('if a > 0')
}

fib(n) {
  if n < 2 {
    return n
  }
  return fib(n - 1) + fib(n - 2)
}

r = fib(1)
echoln(r)
r = fib(2)
echoln(r)
r = fib(3)
echoln(r)
r = fib(4)
echoln(r)
r = fib(5)
echoln(r)
r = fib(6)
echoln(r)
r = fib(7)
echoln(r)
r = fib(8)
echoln(r)

person = {
  name = 'jerloo'
  age = 10
}
assert(person.name == 'jerloo')
echoln(person.age)

Object() {
  return {
    name = 'jerloo'
    age = 10
    isAdult() {
      this.age = this.age + 5
      echoln('this.age ', this.age)
      return true
    }
  }
}

obj = Object()
assert(obj.name == 'jerloo')
obj.age = 20
assert(obj.age == 20)
assert(obj.isAdult())
echoln(obj.isAdult())
echoln(obj.age)

arrdemo = [1,2,3]
echoln(arrdemo[2])
assert(arrdemo[2]==3)

hashTest = 'i am string'
echoln(hashTest)
echoln('hash(i am string) => ', hash(hashTest))

echoln('max(10, 20) => ', max(10,20))

import 'funny.imported.fun'

echoln('uuid => ', uuid())

deepObj = {
  a = {
    b = {
      c = 1
    }
  }
}

echoln('deepObj.a =>', test.a)
echoln('deepObj.a.b =>', test.a.b)
echoln('deepObj.a.b.c =>', test.a.b.c)
```

```console
$ funny --help

usage: funny [<flags>] [<script>]

funny lang

Flags:
  --help    Show context-sensitive help (also try --help-long and --help-man).
  --lexer   tokenizer script
  --parser  parser AST

Args:
  [<script>]  script file path
```

## Todos

- Fix many and many bugs
- Fix scope
- Fix echo
- Add more builtin functions
- Add tests
- ~~Fix import feature~~
- Typings
- module and package feature
- module repo based on github
- Add everything with(have) comment's feature
- Chinese comments length

## License

The MIT License (MIT)

Copyright (c) 2018 jerloo
