all:: run

run:: build
	(cd generated; mvn exec:java)

build:: gen
	(cd generated; mvn compile)

gen: generated generated/src/main/java/CrudlController.java generated/src/main/java/Main.java

generated/src/main/java/CrudlController.java: CrudlController.java
	cp -p CrudlController.java generated/src/main/java/CrudlController.java

generated/src/main/java/Main.java: ../crudl.sadl
	sadl -g java-server -o generated ../crudl.sadl

generated:
	mkdir -p generated/src/main/java

clean::
	rm -rf generated

