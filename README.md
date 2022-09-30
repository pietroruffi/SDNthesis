This repository includes the work i made for my Bachelor Thesis, i.e. implementing of new functionalities in SDN controller defined by Unibo Laboratory of Information and System SEcurity (ULISSe)

# How to run

1) Start mininet
```
make topo
```

2) Start controller (in another terminal)
```
make ctrl
```

3) Use a web browser and connect to `localhost:3333`

## Troubleshooting

minet will read the .p4 file and create files needed by controller (like the p4info.txt) only for the .p4 program specified in Makefile of mininet folder `p4program = ..`

So you have to execute the `make topo` command specifing every time a different p4 program, once for every program you have in p4 folder, to let mininet create all utility files.

# Parser P4

There are two different types of fields, which have to be parsed in different ways: Keys and Action_Parameters, in order to do this we defined some util functions:

## ParseP4Info

Method ParseP4Info accept the name of the name of a p4_program and returns a JSON string cointaining the `[]RuleDescriber` extracted from [P4Info](https://pkg.go.dev/github.com/p4lang/p4runtime@v1.3.0/go/p4/config/v1#P4Info)

The fields extracted from file for every action are:
```
TableName
TableId
Keys ( []FieldDescriber )
ActionName
ActionId
ActionParams ( []FieldDescriber )
```

The struct FieldDescriber contains a *Pattern*, filled using the function `findIfKnownPattern`, which research if the key/parameter respects a known pattern (ex. a 32 bitwidth field with "addr" in his name is recognized as an ipv4_address). Then the Pattern will be used by parsers to know how to properly parse the parameter/key

## ParserMatchInterface

In order to parse Keys, it had been defined an interface ParserMatchInterface, which exposes the method parse(..) for one key, and returns the parsed MatchInterface

Since there are different match types for Keys (ex. EXACT, TERNARY, LPM, ...), every match type has his own parser, which can be obtained by the method `getParserForMatchInterface(parserType string)`, a sort of parser factory, that returns the parser for matchType requested, on which the parse method can be invoked

## ParserActionParams

In order to keep things similar, this parser had been defined like the one for match interfaces, so it has an interface ParserActionParams which exposes the method parse(..), but this time this method parses all action parameters together and returns the corrisponding `[][]byte`

A difference between Action_Parameters and Keys is that Action_Parameters are not influenced by match type; this reflects in the parses because there's no need to define more than one parser for Action_Parameters (it have been defined only one defaultParser, but with this implementation we allow to add new parsers differentiating by some criterious, like the parsers for MatchInterface which are differentiated by matchType)

## How to extend parser.go

If want to add new parser for MatchInterfaces or ActionParameters, need to define a new struct which exposes the method "parse" for the field you choose.

For example a new parser for MatchInterfaces should define:
```
type NewMatchParser struct{}
func (p *NewMatchParser) parse(key Key, describer FieldDescriber) client.MatchInterface { ... }
```
Then in the function parse you need to define the parsing logic for every Pattern that could be parsed

If want to add a new *Pattern* (i.e. description of a field, indicating that it had to be parsed in a specific way), it can be added in the method findIfKnownPattern, and then the parsing logic should be added in every parser that could parse the field
