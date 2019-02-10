package main

import(
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func createPom(name, dir string) error {
	path := filepath.Join(dir, "pom.xml")
	if fileExists(path) {
		fmt.Println("[pom.xml already exists, not overwriting]")
		return nil
	}
   f, err := os.Create(path)
   if err != nil {
		return err
   }
   writer := bufio.NewWriter(f)
	_, err = writer.WriteString(strings.Replace(serverPom, "{{name}}", name, -1)) //ok, this could be generalized to invoking the template engine if I need to...
	writer.Flush()
	f.Close()
	return err
}

const serverPom = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/maven-v4_0_0.xsd">

  <modelVersion>4.0.0</modelVersion>
  
  <groupId>boynton.com</groupId>
  <artifactId>{{name}}</artifactId>
  <packaging>jar</packaging>
  <version>1.0-SNAPSHOT</version>
  <name>{{name}}</name>

  <repositories>
  </repositories>
  
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>org.glassfish.jersey</groupId>
        <artifactId>jersey-bom</artifactId>
        <version>${jersey.version}</version>
        <type>pom</type>
        <scope>import</scope>
      </dependency>
    </dependencies>
  </dependencyManagement>

  <dependencies>    <dependency>
      <groupId>org.glassfish.jersey.containers</groupId>
      <artifactId>jersey-container-jetty-http</artifactId>
    </dependency>
    <dependency>
      <groupId>org.glassfish.jersey.inject</groupId>
      <artifactId>jersey-hk2</artifactId>
    </dependency>
    <dependency>
      <groupId>org.glassfish.jersey.media</groupId>
      <artifactId>jersey-media-json-jackson</artifactId>
    </dependency>

  </dependencies>
  
  <build>
    <plugins>
      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>2.5.1</version>
        <inherited>true</inherited>
        <configuration>
          <source>1.8</source>
          <target>1.8</target>
        </configuration>
      </plugin>
      <plugin>
        <groupId>org.codehaus.mojo</groupId>
        <artifactId>exec-maven-plugin</artifactId>
        <version>1.2.1</version>
        <executions>
          <execution>
            <goals>
              <goal>java</goal>
            </goals>
          </execution>
        </executions>
        <configuration>
          <mainClass>Main</mainClass>
        </configuration>
      </plugin>
    </plugins>
  </build>
  
  <properties>
    <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    <jersey.version>2.27</jersey.version>
  </properties>
</project>

`

