# extract.py

file = open("car2/cons/nfd.log","r")

faces = {}
NLSR = 0
CARI = 0
Data = 0

for line in file.readlines():
    if "onOutgoingInterest" in line and "/ndn/Telkom-University/FTE/Ade" in line:
        print(line)
        face = line.split("out=")[1].split(" ")[0]
        if face in faces.keys():
            faces[face] = faces[face] + 1
        else :
            faces[face] = 1
        Data += 1

    if "onOutgoingInterest" in line:
        if "nlsr" in line:
            NLSR += 1
        
        elif "hello" in line:
            CARI += 1

        if "update" in line:
            CARI += 1
        
        if "info" in line:
            CARI += 1

    if "onIncomingInterest" in line:
        if "nlsr" in line:
            NLSR += 1
        
        elif "hello" in line:
            CARI += 1

        if "update" in line:
            CARI += 1
        
        if "info" in line:
            CARI += 1

print(faces)

print("NLSR : ", (NLSR / Data) * 100)
print("CARI : ", (CARI / Data) * 100)