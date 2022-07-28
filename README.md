# Parser P4
 
 
## Implementation of parser.go

### ParseP4Info(p4Program string) *string

There are two different types of fields, which have to be parsed in different ways: Keys and Action_Parameters

The parser firsty offers method ParseP4Info, which accept the name of the name of p4 program and returns a string cointaining a JSON of []RuleDescriber extracted from the file p4ProgramName.p4.p4info.json

The fields extracted from file are
- TableName
- TableId
- Keys ( []FieldDescriber )
- ActionName
- ActionId
- ActionParams ( []FieldDescriber )

The struct FieldDescriber has a field called Pattern, filled using the function findIfKnownPattern, which research if the field respects a known pattern (es. ipv4_address), then the Pattern will be used from parsers to know how to properly parse parameters/keys

### ParserMatchInterface

In order to parse Keys, it had been defined an interface ParserMatchInterface which exposes the method parse for one key, and returns the corrispondent MatchInterface

To differentiate 

