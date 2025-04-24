import csv
import random

FUNCTION_CODES = [1, 2, 3, 4]

DATA_TYPES = [
    "bitfield",
    "bool",
    "uint8",
    "int8",
    "uint16",
    "int16",
    "uint32",
    "int32",
    "float32",
    "float64",
    "string",
]

DATA_ORDERS = ["A", "AB", "BA", "ABCD", "DCBA", "BADC", "CDAB", "ABCDEFGH", "HGFEDCBA"]


def generate_random_byte_array(length):
    return [random.randint(0, 255) for _ in range(length)]


def generate_csv_data(num_records):
    data = []
    current_address = 1
    for i in range(num_records):
        tag = f"Tag_{i}"
        alias = f"Alias_{i}"
        slaver_id = random.randint(1, 10)
        function = random.choice(FUNCTION_CODES)
        if i % 5 == 0 and i > 0:
            current_address += 2
        else:
            current_address += 1
        read_address = current_address
        data_type = random.choice(DATA_TYPES)
        bit_position = random.randint(0, 15)
        bit_mask = 1 << bit_position

        if data_type in ["uint8", "int8"]:
            valid_orders = ["A"]
            read_quantity = 1
        elif data_type in ["uint16", "int16"]:
            valid_orders = ["AB", "BA"]
            read_quantity = 1
        elif data_type in ["uint32", "int32", "float32"]:
            valid_orders = ["ABCD", "DCBA", "BADC", "CDAB"]
            read_quantity = 2
        elif data_type == "float64":
            valid_orders = ["ABCDEFGH", "HGFEDCBA"]
            read_quantity = 4
        elif data_type == "string":
            valid_orders = ["ABCD"]  # DataOrder is ignored for strings
            read_quantity = random.randint(1, 4)
        else:  # bitfield, bool
            valid_orders = ["AB"]
            read_quantity = 1

        data_order = random.choice(valid_orders)
        weight = round(random.uniform(0.1, 10.0), 3)
        frequency = random.randint(10, 1000)

        record = [
            tag,
            alias,
            slaver_id,
            function,
            read_address,
            read_quantity,
            data_type,
            data_order,
            bit_position,
            bit_mask,
            weight,
            frequency,
        ]
        data.append(record)
    return data


def save_to_csv(data, filename):
    with open(filename, mode="w", newline="") as file:
        writer = csv.writer(file)
        writer.writerow(
            [
                "Tag",
                "Alias",
                "SlaverId",
                "Function",
                "ReadAddress",
                "ReadQuantity",
                "DataType",
                "DataOrder",
                "BitPosition",
                "BitMask",
                "Weight",
                "Frequency",
            ]
        )
        writer.writerows(data)


if __name__ == "__main__":
    num_records = 1000
    csv_data = generate_csv_data(num_records)
    save_to_csv(csv_data, "modbus_registers.csv")
    print(f"Generated {num_records} records and saved to modbus_registers.csv")
