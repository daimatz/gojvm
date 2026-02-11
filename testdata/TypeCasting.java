public class TypeCasting {
    static class Animal {
        String name;
        Animal(String name) { this.name = name; }
        String speak() { return "..."; }
    }
    static class Dog extends Animal {
        Dog(String name) { super(name); }
        String speak() { return "Woof"; }
        String fetch() { return "fetching"; }
    }
    static class Cat extends Animal {
        Cat(String name) { super(name); }
        String speak() { return "Meow"; }
    }

    public static void main(String[] args) {
        Animal a = new Dog("Rex");
        System.out.println(a.speak());          // Woof (virtual dispatch)
        System.out.println(a instanceof Dog);   // true
        System.out.println(a instanceof Cat);   // false
        System.out.println(a instanceof Animal);// true

        // downcast
        Dog d = (Dog) a;
        System.out.println(d.fetch());          // fetching

        // Object array with mixed types
        Object[] objs = new Object[3];
        objs[0] = "hello";
        objs[1] = a;
        objs[2] = new Cat("Whiskers");
        for (Object o : objs) {
            if (o instanceof String) {
                System.out.println("String: " + (String) o);
            } else if (o instanceof Dog) {
                System.out.println("Dog: " + ((Dog) o).name);
            } else if (o instanceof Cat) {
                System.out.println("Cat: " + ((Cat) o).name);
            }
        }
    }
}
