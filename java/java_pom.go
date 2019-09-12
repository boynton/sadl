package java

import (
	"fmt"
	"path/filepath"
	"text/template"
)

func (gen *Generator) CreatePom(domain, name, dir string, lombok bool, extraDepends string) {
	path := filepath.Join(dir, "pom.xml")
	if gen.FileExists(path) {
		fmt.Println("[pom.xml already exists, not overwriting]")
		return
	}
	/*	f, err := os.Create(path)
		if err != nil {
			return err
		}
		writer := bufio.NewWriter(f)
	*/
	dependsMgt := jerseyDependsMgt
	depends := jerseyDepends
	versions := jerseyVersion
	if lombok {
		depends = depends + lombokDepends
	}
	if extraDepends != "" {
		depends = depends + extraDepends
	}
	funcMap := template.FuncMap{
		"domain":     func() string { return domain },
		"name":       func() string { return name },
		"dependsMgt": func() string { return dependsMgt },
		"depends":    func() string { return depends },
		"versions":   func() string { return versions },
	}
	gen.Begin()
	gen.EmitTemplate("pom.xml", pomTemplate, gen, funcMap)
	result := gen.End()
	if gen.Err == nil {
		gen.WriteFile(path, result)
	}
	/*
		s := pomTemplate
		s = strings.Replace(s, "{{domain}}", domain, -1)
		s = strings.Replace(s, "{{name}}", name, -1)
		s = strings.Replace(s, "{{dependsMgt}}", dependsMgt, -1)
		s = strings.Replace(s, "{{depends}}", depends, -1)
		s = strings.Replace(s, "{{versions}}", versions, -1)
		_, err = writer.WriteString(s)
		writer.Flush()
		f.Close()
		return err
	*/
}

const jerseyVersion = `    <jersey.version>2.28</jersey.version>                                                                          `

const jerseyDependsMgt = `  <dependencyManagement>
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
`

const jerseyDepends = `    <dependency>
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
<!-- the following is needed to compile with java11 -->
    <dependency>
      <groupId>javax.activation</groupId>
      <artifactId>activation</artifactId>
      <version>1.1</version>
    </dependency>
    <dependency>
      <groupId>javax.xml.bind</groupId>
      <artifactId>jaxb-api</artifactId>
      <version>2.3.0</version>
    </dependency>
`

const lombokDepends = `      <dependency>
        <groupId>org.projectlombok</groupId>
        <artifactId>lombok</artifactId>
        <version>1.18.6</version>
        <scope>provided</scope>
      </dependency>
`

const pomTemplate = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/maven-v4_0_0.xsd">

  <modelVersion>4.0.0</modelVersion>
  
  <groupId>{{domain}}</groupId>
  <artifactId>{{name}}</artifactId>
  <packaging>jar</packaging>
  <version>1.0-SNAPSHOT</version>
  <name>{{name}}</name>

{{dependsMgt}}
  <dependencies>
{{depends}}
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
{{versions}}
  </properties>
</project>
`
