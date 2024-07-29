# Speedtest-BLE-PC 

PC software for [Speedtest](https://github.com/Nigh/Speedtest-nRF52832) firmware

## Usage

```
go run ./main.go
```

## Example Result

```
enabling
scanning...
found device: D7:9D:F5:DC:85:AB Speedtest -26
connected to  D7:9D:F5:DC:85:AB
discovering services/characteristics
found service 0000a800-0000-1000-8000-00805f9b34fb
found characteristic 0000a801-0000-1000-8000-00805f9b34fb 0000a802-0000-1000-8000-00805f9b34fb
Test will start after 3 seconds
3
2
1
package:    0 length:244
package:    1 length:244
package:    2 length:244

...

package: 2094 length:244
package: 2096 length:244
package: 2095 length:244
package: 2097 length:244
Test complete
Transfer 2098 packages(511912 bytes) in 5876ms
0 packages loss []
0 packages repeat []
Average speed is 87.11 kB/s
```
