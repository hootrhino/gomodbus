import csv
import random

FUNCTION_CODES = [1, 2, 3, 4]

DATA_TYPES = ["int16", "int32", "float32"]

DATA_ORDERS = ["AB", "BA", "ABCD", "DCBA", "CDAB", "BADC"]

def generate_random_byte_array():
    return [random.randint(0, 255) for _ in range(8)]

def generate_csv_data(num_records):
    data = []
    current_address = 1
    for i in range(num_records):
        tag = f"Tag_{i}"
        alias = f"Alias_{i}"
        slaver_id = 1
        function = 3
        if i % 5 == 0 and i > 0:
            current_address += 2
        else:
            current_address += 1
        read_address = current_address
        data_type = random.choice(DATA_TYPES)
        bit_position = 1
        bit_mask = 1

        if data_type == "int16":
            valid_orders = ["AB", "BA"]
            read_quantity = 1
        else:
            valid_orders = ["ABCD", "DCBA", "CDAB", "BADC"]
            read_quantity = 2
        data_order = random.choice(valid_orders)

        if i % 5 == 0 and i > 0:
            weight = 1
        else:
            weight = 3.141
        frequency = 50

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
            frequency
        ]
        data.append(record)
    return data

def save_to_csv(data, filename):
    with open(filename, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow([
            "Tag", "Alias", "SlaverId", "Function", "ReadAddress",
            "ReadQuantity", "DataType", "DataOrder", "BitPosition",
            "BitMask", "Weight", "Frequency"
        ])
        writer.writerows(data)

if __name__ == "__main__":
    num_records = 100
    csv_data = generate_csv_data(num_records)
    save_to_csv(csv_data, "modbus_registers.csv")
