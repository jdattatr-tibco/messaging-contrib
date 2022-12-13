
# Apache Pulsar Trigger
This trigger provides your flogo application the ability to build pulsar functions.

## Installation

```bash
flogo install github.com/jdattatr-tibco/messaging-contrib/pulsar/trigger/function
```
## Configuration

### Output:
| Name        | Type   | Description
|:---         | :---   | :---        
| message     | bytes  | The message from the Pulsar Queue.

### Reply:
| Name        | Type   | Description
|:---         | :---   | :---        
| out         | any    | The output from flogo action.