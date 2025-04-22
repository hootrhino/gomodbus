import logging
from pymodbus.server import StartTcpServer
from pymodbus.datastore import ModbusSequentialDataBlock
from pymodbus.datastore import ModbusSlaveContext, ModbusServerContext

# Enable logging mode
FORMAT = (
    "%(asctime)-15s %(threadName)-15s "
    "%(levelname)-8s %(module)-15s:%(lineno)-8s %(message)s"
)
logging.basicConfig(format=FORMAT)
log = logging.getLogger()
log.setLevel(logging.DEBUG)


def run_modbus_server():
    # Initialize coil data block, set 1000 coil values to True (ON)
    coils = [True] * 1000
    coil_block = ModbusSequentialDataBlock(0, coils)

    # Initialize holding register data block, set 1000 register values to 0xFFFF
    registers = [0xFFFF] * 1000
    register_block = ModbusSequentialDataBlock(0, registers)

    # Create slave context
    store = ModbusSlaveContext(
        di=coil_block, co=coil_block, hr=register_block, ir=register_block
    )
    context = ModbusServerContext(slaves=store, single=True)

    # Start TCP server, listen on port 5020
    StartTcpServer(context, address=("localhost", 5020))


if __name__ == "__main__":
    run_modbus_server()
