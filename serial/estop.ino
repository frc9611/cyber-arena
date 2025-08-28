void setup() {
  pinMode(8, INPUT_PULLUP);
  Serial.begin(9600);
}

void loop() {
  if(Serial.available() > 0) {
    Serial.print("pong");
    Serial.readString();
  }

  bool btnPressed = !digitalRead(8);
  if(btnPressed) {
    Serial.print(0x9);
  }else{
    Serial.println(0xf);
  }
  delay(200);
}
