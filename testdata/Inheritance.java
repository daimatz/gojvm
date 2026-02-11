abstract class Animal {
    abstract int sound();
    int speak() { return sound(); }
}
class Dog extends Animal {
    int sound() { return 1; }
}
class Cat extends Animal {
    int sound() { return 2; }
}
class Inheritance {
    static int check(Animal a) { return a.speak(); }
    public static void main(String[] args) {
        System.out.println(check(new Dog()));
        System.out.println(check(new Cat()));
    }
}
